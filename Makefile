.PHONY: up down rebuild logs clean \
        api-build api-run api-tidy \
        front-install front-dev front-build front-lint \
        db-migrate db-reset db-psql \
        doc doc-task doc-clean \
        help

# ============================================================
#  docker compose
# ============================================================

up:
	docker compose up --build -d

down:
	docker compose down

rebuild:
	docker compose down
	docker compose up --build -d

logs:
	docker compose logs -f

logs-api:
	docker compose logs -f api

logs-front:
	docker compose logs -f front

clean:
	docker compose down -v

# ============================================================
#  api (go)
# ============================================================

api-build:
	cd plantation-api && go build -o bin/api ./cmd/api

api-run:
	cd plantation-api && go run ./cmd/api

api-tidy:
	cd plantation-api && go mod tidy

# ============================================================
#  frontend (node/vite)
# ============================================================

front-install:
	cd plantation-front && npm install

front-dev:
	cd plantation-front && npm run dev

front-build:
	cd plantation-front && npm run build

front-lint:
	cd plantation-front && npm run lint

# ============================================================
#  database
# ============================================================

db-migrate:
	docker compose exec db psql -U postgres -d plantation \
		-f /docker-entrypoint-initdb.d/001_init.sql
	docker compose exec db psql -U postgres -d plantation \
		-f /docker-entrypoint-initdb.d/002_simulation.sql

db-reset:
	docker compose down -v
	docker compose up -d db
	@echo "waiting for postgres..."
	@sleep 3
	docker compose up -d api front

db-psql:
	docker compose exec db psql -U postgres -d plantation

# ============================================================
#  documentation (latex)
# ============================================================

DOC_DIR   = doc
LATEX     = pdflatex
BIBTEX    = bibtex
TEX_FLAGS = -interaction=nonstopmode -halt-on-error

doc: doc-task
	cd $(DOC_DIR) && $(LATEX) $(TEX_FLAGS) paper.tex
	cd $(DOC_DIR) && $(BIBTEX) paper
	cd $(DOC_DIR) && $(LATEX) $(TEX_FLAGS) paper.tex
	cd $(DOC_DIR) && $(LATEX) $(TEX_FLAGS) paper.tex

doc-task:
	cd $(DOC_DIR) && $(LATEX) $(TEX_FLAGS) task.tex

doc-clean:
	cd $(DOC_DIR) && rm -f *.aux *.log *.out *.toc *.bbl *.blg *.run.xml *.bcf

# ============================================================
#  help
# ============================================================

help:
	@echo ""
	@echo "  docker"
	@echo "    make up / down / rebuild / logs / clean"
	@echo "    make logs-api          only api logs"
	@echo "    make logs-front        only frontend logs"
	@echo ""
	@echo "  api"
	@echo "    make api-build / api-run / api-tidy"
	@echo ""
	@echo "  frontend"
	@echo "    make front-install / front-dev / front-build / front-lint"
	@echo ""
	@echo "  database"
	@echo "    make db-migrate / db-reset / db-psql"
	@echo ""
	@echo "  documentation"
	@echo "    make doc             build paper.pdf (with bibtex)"
	@echo "    make doc-task        build task.pdf only"
	@echo "    make doc-clean       remove latex artifacts"
	@echo ""