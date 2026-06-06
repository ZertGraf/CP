package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"plantation-api/internal/model"
)

type Storage struct {
	DB *bun.DB
}

func New(dsn string) *Storage {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())
	return &Storage{DB: db}
}

// --- users ---

func (s *Storage) CreateUser(ctx context.Context, u *model.User) error {
	_, err := s.DB.NewInsert().Model(u).Exec(ctx)
	return err
}

func (s *Storage) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	u := new(model.User)
	err := s.DB.NewSelect().Model(u).Where("email = ?", email).Scan(ctx)
	return u, err
}

// --- sectors ---

func (s *Storage) ListSectors(ctx context.Context) ([]model.Sector, error) {
	var sectors []model.Sector
	err := s.DB.NewSelect().Model(&sectors).Order("created_at DESC").Scan(ctx)
	return sectors, err
}

func (s *Storage) GetSector(ctx context.Context, id string) (*model.Sector, error) {
	sector := new(model.Sector)
	err := s.DB.NewSelect().Model(sector).Where("id = ?", id).Scan(ctx)
	return sector, err
}

func (s *Storage) CreateSector(ctx context.Context, sec *model.Sector) error {
	_, err := s.DB.NewInsert().Model(sec).Exec(ctx)
	return err
}

func (s *Storage) UpdateSector(ctx context.Context, sec *model.Sector) error {
	_, err := s.DB.NewUpdate().Model(sec).WherePK().Exec(ctx)
	return err
}

func (s *Storage) DeleteSector(ctx context.Context, id string) error {
	_, err := s.DB.NewDelete().Model((*model.Sector)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

// --- plants ---

func (s *Storage) ListPlants(ctx context.Context, sectorID string) ([]model.Plant, error) {
	var plants []model.Plant
	q := s.DB.NewSelect().Model(&plants).Order("created_at DESC")
	if sectorID != "" {
		q = q.Where("sector_id = ?", sectorID)
	}
	err := q.Scan(ctx)
	return plants, err
}

func (s *Storage) CreatePlant(ctx context.Context, p *model.Plant) error {
	_, err := s.DB.NewInsert().Model(p).Exec(ctx)
	return err
}

func (s *Storage) DeletePlant(ctx context.Context, id string) error {
	_, err := s.DB.NewDelete().Model((*model.Plant)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

// --- watering ---

func (s *Storage) CreateWateringLog(ctx context.Context, log *model.WateringLog) error {
	_, err := s.DB.NewInsert().Model(log).Exec(ctx)
	return err
}

func (s *Storage) GetWateringStats(ctx context.Context, sectorID string) (float64, int, error) {
	var result struct {
		Total float64 `bun:"total"`
		Count int     `bun:"count"`
	}
	err := s.DB.NewSelect().
		TableExpr("watering_logs").
		ColumnExpr("COALESCE(SUM(volume_liters), 0) AS total").
		ColumnExpr("COUNT(*) AS count").
		Where("sector_id = ?", sectorID).
		Scan(ctx, &result)
	return result.Total, result.Count, err
}

// --- telemetry ---

func (s *Storage) SaveTelemetry(ctx context.Context, t *model.Telemetry) error {
	_, err := s.DB.NewInsert().Model(t).Exec(ctx)
	return err
}

func (s *Storage) ListTelemetry(ctx context.Context, sectorID string, limit int) ([]model.Telemetry, error) {
	var rows []model.Telemetry
	q := s.DB.NewSelect().Model(&rows).Order("recorded_at DESC").Limit(limit)
	if sectorID != "" {
		q = q.Where("sector_id = ?", sectorID)
	}
	err := q.Scan(ctx)
	return rows, err
}

func (s *Storage) GetTelemetryReport(ctx context.Context, sectorID string) (avgMoisture, avgTemp, minHealth float64, count int, err error) {
	var result struct {
		AvgMoisture float64 `bun:"avg_m"`
		AvgTemp     float64 `bun:"avg_t"`
		MinHealth   float64 `bun:"min_h"`
		Count       int     `bun:"cnt"`
	}
	err = s.DB.NewSelect().
		TableExpr("telemetry").
		ColumnExpr("COALESCE(AVG(soil_moisture), 0) AS avg_m").
		ColumnExpr("COALESCE(AVG(temperature), 0) AS avg_t").
		ColumnExpr("COALESCE(MIN(health_index), 0) AS min_h").
		ColumnExpr("COUNT(*) AS cnt").
		Where("sector_id = ?", sectorID).
		Scan(ctx, &result)
	return result.AvgMoisture, result.AvgTemp, result.MinHealth, result.Count, err
}

// --- notifications ---

func (s *Storage) CreateNotification(ctx context.Context, n *model.Notification) error {
	_, err := s.DB.NewInsert().Model(n).Exec(ctx)
	return err
}

func (s *Storage) ListNotifications(ctx context.Context, userID string, onlyUnread bool) ([]model.Notification, error) {
	var rows []model.Notification
	q := s.DB.NewSelect().Model(&rows).Order("created_at DESC").Limit(50)
	if onlyUnread {
		q = q.Where("is_read = FALSE")
	}
	err := q.Scan(ctx)
	return rows, err
}

func (s *Storage) MarkNotificationRead(ctx context.Context, id string) error {
	_, err := s.DB.NewUpdate().
		Model((*model.Notification)(nil)).
		Set("is_read = TRUE").
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (s *Storage) FixWaterOverflow(ctx context.Context) error {
	_, err := s.DB.NewUpdate().
		Model((*model.Sector)(nil)).
		Set("water_consumed = 0").
		Where("water_consumed > daily_water_limit").
		Exec(ctx)
	return err
}

func (s *Storage) ResetAllWaterConsumed(ctx context.Context) error {
	_, err := s.DB.NewUpdate().
		Model((*model.Sector)(nil)).
		Set("water_consumed = 0").
		Where("TRUE").
		Exec(ctx)
	return err
}

func (s *Storage) AssignOperator(ctx context.Context, sectorID, operatorID string) error {
	_, err := s.DB.NewUpdate().
		Model((*model.Sector)(nil)).
		Set("operator_id = ?", operatorID).
		Set("updated_at = now()").
		Where("id = ?", sectorID).
		Exec(ctx)
	return err
}

func (s *Storage) UnassignOperator(ctx context.Context, sectorID string) error {
	_, err := s.DB.NewUpdate().
		Model((*model.Sector)(nil)).
		Set("operator_id = NULL").
		Set("updated_at = now()").
		Where("id = ?", sectorID).
		Exec(ctx)
	return err
}

func (s *Storage) ListSectorsByOperator(ctx context.Context, operatorID string) ([]model.Sector, error) {
	var sectors []model.Sector
	err := s.DB.NewSelect().
		Model(&sectors).
		Where("operator_id = ?", operatorID).
		Order("created_at DESC").
		Scan(ctx)
	return sectors, err
}

func (s *Storage) GetUser(ctx context.Context, id string) (*model.User, error) {
	u := new(model.User)
	err := s.DB.NewSelect().Model(u).Where("id = ?", id).Scan(ctx)
	return u, err
}

// --- schema bootstrap (idempotent) ---

// EnsureSchema applies the chapter-2 additions (CWSI, streaks, equipment lock,
// training_scores, weather_configs) on top of an existing database. It is safe to
// run on every startup and complements the numbered SQL migration files.
func (s *Storage) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`ALTER TABLE sectors ADD COLUMN IF NOT EXISTS cwsi FLOAT NOT NULL DEFAULT 0`,
		`ALTER TABLE sectors ADD COLUMN IF NOT EXISTS healthy_streak INT NOT NULL DEFAULT 0`,
		`ALTER TABLE sectors ADD COLUMN IF NOT EXISTS safe_streak INT NOT NULL DEFAULT 0`,
		`ALTER TABLE sectors ADD COLUMN IF NOT EXISTS crisis_streak INT NOT NULL DEFAULT 0`,
		`ALTER TABLE sectors ADD COLUMN IF NOT EXISTS equipment_locked_ticks INT NOT NULL DEFAULT 0`,
		`ALTER TABLE sectors ADD COLUMN IF NOT EXISTS pest_active BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE sectors ADD COLUMN IF NOT EXISTS last_alert_kind VARCHAR(50) NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS training_scores (
			id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id          UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
			session_id       UUID,
			total_score      FLOAT NOT NULL DEFAULT 0,
			badges           JSONB NOT NULL DEFAULT '[]',
			avg_health       FLOAT NOT NULL DEFAULT 0,
			water_efficiency FLOAT NOT NULL DEFAULT 0,
			sum_health       FLOAT NOT NULL DEFAULT 0,
			sum_efficiency   FLOAT NOT NULL DEFAULT 0,
			tick_count       INT NOT NULL DEFAULT 0,
			updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS weather_configs (
			id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name         VARCHAR(255) NOT NULL DEFAULT 'default',
			is_active    BOOLEAN NOT NULL DEFAULT TRUE,
			p_dry_to_wet FLOAT NOT NULL DEFAULT 0.20,
			p_wet_to_wet FLOAT NOT NULL DEFAULT 0.55,
			gamma_shape  FLOAT NOT NULL DEFAULT 1.5,
			gamma_scale  FLOAT NOT NULL DEFAULT 6.0,
			p_heat       FLOAT NOT NULL DEFAULT 0.05,
			p_pest_base  FLOAT NOT NULL DEFAULT 0.02,
			p_equipment  FLOAT NOT NULL DEFAULT 0.02,
			latitude     FLOAT NOT NULL DEFAULT 15.47,
			et_method    VARCHAR(20) NOT NULL DEFAULT 'hargreaves',
			created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	}
	for _, q := range stmts {
		if _, err := s.DB.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

// --- weather configuration ---

// GetActiveWeatherConfig returns the active profile, creating the calibrated
// default (Guatemala/Cobán) on first call.
func (s *Storage) GetActiveWeatherConfig(ctx context.Context) (*model.WeatherConfig, error) {
	cfg := new(model.WeatherConfig)
	err := s.DB.NewSelect().Model(cfg).Where("is_active = TRUE").Order("updated_at DESC").Limit(1).Scan(ctx)
	if err != nil {
		cfg = model.DefaultWeatherConfig()
		if _, ierr := s.DB.NewInsert().Model(cfg).Exec(ctx); ierr != nil {
			return model.DefaultWeatherConfig(), nil
		}
	}
	return cfg, nil
}

func (s *Storage) UpdateWeatherConfig(ctx context.Context, cfg *model.WeatherConfig) error {
	cfg.UpdatedAt = time.Now()
	_, err := s.DB.NewUpdate().Model(cfg).WherePK().Exec(ctx)
	return err
}

// --- training scores / leaderboard ---

func (s *Storage) GetTrainingScore(ctx context.Context, userID string) (*model.TrainingScore, error) {
	ts := new(model.TrainingScore)
	err := s.DB.NewSelect().Model(ts).Where("user_id = ?", userID).Scan(ctx)
	return ts, err
}

// ApplyScore upserts the running training-analytics row for an operator: it adds the
// per-tick score, updates the running averages of health and water efficiency, and
// merges any newly earned badges (chapter 2.4).
func (s *Storage) ApplyScore(ctx context.Context, userID, sessionID string, delta, health, efficiency float64, newBadges []string) error {
	ts, err := s.GetTrainingScore(ctx, userID)
	if err != nil {
		ts = &model.TrainingScore{
			UserID:    userID,
			SessionID: sessionID,
			Badges:    []string{},
		}
	}

	ts.TotalScore += delta
	ts.SumHealth += health
	ts.SumEfficiency += efficiency
	ts.TickCount++
	if ts.TickCount > 0 {
		ts.AvgHealth = ts.SumHealth / float64(ts.TickCount)
		ts.WaterEfficiency = ts.SumEfficiency / float64(ts.TickCount)
	}
	ts.Badges = mergeBadges(ts.Badges, newBadges)
	ts.UpdatedAt = time.Now()

	if ts.ID == "" {
		_, err = s.DB.NewInsert().Model(ts).Exec(ctx)
	} else {
		_, err = s.DB.NewUpdate().Model(ts).WherePK().Exec(ctx)
	}
	return err
}

// AwardScore lets the agronomist manually adjust an operator's points and badges
// (chapter 2.4). Creates the analytics row if the operator has none yet.
func (s *Storage) AwardScore(ctx context.Context, userID string, points float64, addBadges, removeBadges []string) (*model.TrainingScore, error) {
	ts, err := s.GetTrainingScore(ctx, userID)
	if err != nil {
		ts = &model.TrainingScore{UserID: userID, Badges: []string{}}
	}

	ts.TotalScore += points
	ts.Badges = mergeBadges(ts.Badges, addBadges)
	if len(removeBadges) > 0 {
		ts.Badges = removeBadgesFrom(ts.Badges, removeBadges)
	}
	if ts.Badges == nil {
		ts.Badges = []string{}
	}
	ts.UpdatedAt = time.Now()

	if ts.ID == "" {
		_, err = s.DB.NewInsert().Model(ts).Exec(ctx)
	} else {
		_, err = s.DB.NewUpdate().Model(ts).WherePK().Exec(ctx)
	}
	return ts, err
}

func removeBadgesFrom(existing, remove []string) []string {
	drop := make(map[string]bool, len(remove))
	for _, b := range remove {
		drop[b] = true
	}
	out := make([]string, 0, len(existing))
	for _, b := range existing {
		if !drop[b] {
			out = append(out, b)
		}
	}
	return out
}

func mergeBadges(existing, incoming []string) []string {
	seen := make(map[string]bool, len(existing))
	out := make([]string, 0, len(existing)+len(incoming))
	for _, b := range existing {
		if !seen[b] {
			seen[b] = true
			out = append(out, b)
		}
	}
	for _, b := range incoming {
		if !seen[b] {
			seen[b] = true
			out = append(out, b)
		}
	}
	return out
}

// Leaderboard returns operators ranked by total score for the agronomist view.
func (s *Storage) Leaderboard(ctx context.Context) ([]model.LeaderboardEntry, error) {
	var rows []model.TrainingScore
	err := s.DB.NewSelect().
		Model(&rows).
		ModelTableExpr("training_scores AS ts").
		ColumnExpr("ts.*").
		ColumnExpr("u.name AS user_name").
		Join("JOIN users u ON u.id = ts.user_id").
		Order("ts.total_score DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]model.LeaderboardEntry, 0, len(rows))
	for _, r := range rows {
		badges := r.Badges
		if badges == nil {
			badges = []string{}
		}
		out = append(out, model.LeaderboardEntry{
			UserID:          r.UserID,
			Name:            r.UserName,
			TotalScore:      r.TotalScore,
			AvgHealth:       r.AvgHealth,
			WaterEfficiency: r.WaterEfficiency,
			Badges:          badges,
		})
	}
	return out, nil
}
