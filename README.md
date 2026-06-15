# Plantation — тренажёр управления орошением плантации

Веб-симулятор для обучения операторов агрономии: система моделирует
жизнь секторов плантации личи в реальном времени (рост, влагозапас почвы,
погодные события, вредители, отказы оборудования), а пользователь
управляет поливом и реагирует на инциденты. Действия оцениваются —
поверх симуляции построена геймификация (баллы, значки, таблица лидеров).

Проект выполнен как курсовая работа; пояснительная записка (LaTeX)
лежит в [`doc/`](doc/).

## Стек

| Слой | Технологии |
|------|-----------|
| Backend | Go 1.23, [chi](https://github.com/go-chi/chi) router, [bun](https://bun.uptrace.dev) ORM, [gorilla/websocket](https://github.com/gorilla/websocket), JWT (golang-jwt v5) |
| Frontend | React 19, Vite 8 (SPA), WebSocket для real-time телеметрии |
| База данных | PostgreSQL 16 |
| Инфраструктура | Docker Compose, nginx (раздача статики фронта), Caddy (обратный прокси на хосте), GitHub Actions (CI/CD) |

## Архитектура

```
              ┌─────────┐        /api, /ws         ┌──────────┐
  браузер ──► │  Caddy  │ ───────────────────────► │   api    │ ──► PostgreSQL
              │ (хост)  │                           │ (Go)     │ ◄── симуляция (тики)
              └─────────┘ ──► front (nginx, SPA)    └──────────┘
```

- **api** — REST + WebSocket. При старте применяет миграции (`EnsureSchema`)
  и засевает тестовые данные, если БД пуста. Внутри крутится
  движок симуляции, который тиками обновляет состояние секторов и
  рассылает телеметрию/уведомления через WebSocket-хаб.
- **front** — статический бандл React, который nginx раздаёт как SPA.
  Все запросы к `/api` и `/ws` проксирует Caddy (в dev — встроенный
  прокси Vite на `api:8080`).
- **db** — PostgreSQL; том `pg_data` для персистентности, SQL-миграции
  монтируются в `docker-entrypoint-initdb.d`.

Сервисы `api` и `front` не публикуют порты наружу — наружу смотрит
только Caddy через внешнюю сеть `caddy_default`.

## Структура репозитория

```
plantation-api/          backend на Go
  cmd/api/               точка входа
  internal/
    handler/             HTTP-обработчики (auth, sectors, plants, water, ...)
    middleware/          JWT-аутентификация, проверка ролей
    model/               доменные модели
    simulation/          движок симуляции, генератор погоды, агромодель
    storage/             доступ к БД (bun), сидинг, EnsureSchema
    ws/                  WebSocket-хаб
    xlsx/                импорт/экспорт XLSX
  migrations/            001_init, 002_simulation, 003_gamification
plantation-front/        SPA на React + Vite
docker-compose.yml       db + api + front
Makefile                 команды для разработки и сборки
doc/                     пояснительная записка (LaTeX)
.github/workflows/       деплой по push в main
```

## Быстрый старт (Docker)

Compose рассчитан на работу за обратным прокси Caddy и использует
внешнюю сеть `caddy_default`. Создайте её один раз (или поднимите Caddy,
который создаст её сам):

```bash
docker network create caddy_default
make up            # docker compose up --build -d
make logs          # хвост логов
make down          # остановить
```

Полезные переменные окружения (можно задать в `.env` рядом с compose):

| Переменная | Назначение | По умолчанию |
|------------|-----------|--------------|
| `POSTGRES_PASSWORD` | пароль PostgreSQL | `postgres` |
| `JWT_SECRET` | секрет для подписи JWT | `change-me-in-production` |

## Локальная разработка (без Docker)

Нужны установленные Go 1.23+, Node 20+ и доступный PostgreSQL.

```bash
# backend (читает plantation-api/.env: DATABASE_URL, JWT_SECRET, PORT)
make api-run                 # go run ./cmd/api  → http://localhost:8080

# frontend (Vite проксирует /api и /ws на api:8080)
make front-install
make front-dev               # http://localhost:5173
```

При первом запуске api сам создаёт схему и засевает тестовые
учётные записи (агроном и операторы) — регистрировать вручную не нужно.

## Роли и доступ

Аутентификация по JWT, две роли (enum `user_role`):

- **agronomist** — создаёт и настраивает секторы, назначает операторов,
  ручной `override` показателей, конфигурирует генератор погоды,
  начисляет баллы, импорт/экспорт XLSX.
- **operator** — полив секторов, обработка от вредителей, просмотр
  телеметрии, отчётов и своего рейтинга.

## Основные эндпоинты

Все защищённые маршруты требуют заголовок `Authorization: Bearer <token>`.

```
POST   /api/auth/register | /api/auth/login         регистрация / вход
GET    /api/sectors | /api/sectors/{id} | /sectors/my   список / карточка / свои
POST   /api/sectors                                  создать сектор (agronomist)
PUT    /api/sectors/{id}/assign                      назначить оператора (agronomist)
POST   /api/water                                    полить сектор
POST   /api/sectors/{id}/treat                       обработать от вредителей
GET    /api/telemetry/{sectorId} | /api/reports/{id} телеметрия / сводный отчёт
GET    /api/notifications                            уведомления
GET    /api/leaderboard | /api/score/my              геймификация
GET    /api/weather-config (PUT)                     профиль генератора погоды (agronomist)
GET    /api/export/sectors | /api/import/sectors     XLSX-обмен (agronomist)
WS     /ws                                           поток телеметрии и событий
```

## Модель симуляции (кратко)

Движок тиками продвигает агрономическое состояние каждого сектора:

- **рост** — накопление GDD (growing degree days) и переход по фенофазам;
- **влага** — водный баланс почвы, дефицит `Dr`, коэффициенты стресса
  `Ks` (водный и аэрационный), индекс водного стресса CWSI;
- **погода** — марковская цепь сухо/дождь, осадки по гамма-распределению,
  эвапотранспирация (метод Харгривса); вероятности жары, нашествия
  вредителей и отказа оборудования настраиваются через `weather_configs`;
- **оценка** — здоровье растений и эффективность полива агрегируются в
  `training_scores` (баллы, значки, серии без инцидентов).

## Деплой

Push в `main` запускает workflow [`.github/workflows/deploy.yml`](.github/workflows/deploy.yml):
он по SSH синхронизирует проект на сервер (`rsync`) и пересобирает
контейнеры (`docker compose up -d --build`).

Требуются секреты репозитория: `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_SSH_KEY`.
Запустить вручную — `gh workflow run deploy.yml` или из вкладки Actions.

## Документация

Пояснительная записка собирается из LaTeX:

```bash
make doc          # paper.pdf (с bibtex)
make doc-task     # task.pdf
make doc-clean    # удалить промежуточные файлы LaTeX
```

Полный список команд — `make help`.
