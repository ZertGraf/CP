package storage

import (
	"context"
	"database/sql"

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
