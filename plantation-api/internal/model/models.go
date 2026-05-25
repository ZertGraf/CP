package model

import (
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

// users

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID        string    `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	Email     string    `bun:"email,notnull,unique"                      json:"email"`
	Password  string    `bun:"password,notnull"                          json:"-"`
	Name      string    `bun:"name,notnull"                              json:"name"`
	Role      string    `bun:"role,notnull,default:'operator'"           json:"role"`
	CreatedAt time.Time `bun:"created_at,default:now()"                  json:"created_at"`
}

type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

func (r *RegisterInput) Validate() error {
	if !strings.Contains(r.Email, "@") || len(r.Email) < 5 {
		return errors.New("invalid email")
	}
	if len(r.Password) < 6 {
		return errors.New("password must be at least 6 chars")
	}
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if r.Role != "agronomist" && r.Role != "operator" {
		r.Role = "operator"
	}
	return nil
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// sectors

type Sector struct {
	bun.BaseModel `bun:"table:sectors"`

	ID              string     `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	Name            string     `bun:"name,notnull"                              json:"name"`
	AreaSqm         float64    `bun:"area_sqm"                                  json:"area_sqm"`
	SoilMoisture    float64    `bun:"soil_moisture"                             json:"soil_moisture"`
	Temperature     float64    `bun:"temperature"                               json:"temperature"`
	HealthIndex     float64    `bun:"health_index"                              json:"health_index"`
	GddCumulative   float64    `bun:"gdd_cumulative"                            json:"gdd_cumulative"`
	Phenophase      string     `bun:"phenophase"                                json:"phenophase"`
	KsWater         float64    `bun:"ks_water"                                  json:"ks_water"`
	KsAeration      float64    `bun:"ks_aeration"                               json:"ks_aeration"`
	DeficitDr       float64    `bun:"deficit_dr"                                json:"deficit_dr"`
	Status          string     `bun:"status"                                    json:"status"`
	OperatorID      *string    `bun:"operator_id,type:uuid"                     json:"operator_id"`
	LastWateredAt   *time.Time `bun:"last_watered_at"                           json:"last_watered_at"`
	DailyWaterLimit float64    `bun:"daily_water_limit"                         json:"daily_water_limit"`
	WaterConsumed   float64    `bun:"water_consumed"                            json:"water_consumed"`
	CreatedAt       time.Time  `bun:"created_at,default:now()"                  json:"created_at"`
	UpdatedAt       time.Time  `bun:"updated_at,default:now()"                  json:"updated_at"`
}

type SectorInput struct {
	Name    string  `json:"name"`
	AreaSqm float64 `json:"area_sqm"`
}

func (s *SectorInput) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("name is required")
	}
	if s.AreaSqm < 0 {
		return errors.New("area must be positive")
	}
	return nil
}

// plants

type Plant struct {
	bun.BaseModel `bun:"table:plants"`

	ID        string    `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	SectorID  string    `bun:"sector_id,notnull,type:uuid"               json:"sector_id"`
	Species   string    `bun:"species"                                   json:"species"`
	AgeMonths int       `bun:"age_months"                                json:"age_months"`
	Health    int       `bun:"health"                                    json:"health"`
	CreatedAt time.Time `bun:"created_at,default:now()"                  json:"created_at"`
}

type PlantInput struct {
	SectorID  string `json:"sector_id"`
	Species   string `json:"species"`
	AgeMonths int    `json:"age_months"`
}

func (p *PlantInput) Validate() error {
	if strings.TrimSpace(p.SectorID) == "" {
		return errors.New("sector_id is required")
	}
	if strings.TrimSpace(p.Species) == "" {
		p.Species = "litchi"
	}
	if p.AgeMonths < 0 {
		return errors.New("age must be positive")
	}
	return nil
}

// watering

type WateringLog struct {
	bun.BaseModel `bun:"table:watering_logs"`

	ID           string    `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	SectorID     string    `bun:"sector_id,notnull,type:uuid"               json:"sector_id"`
	UserID       string    `bun:"user_id,notnull,type:uuid"                 json:"user_id"`
	VolumeLiters float64   `bun:"volume_liters,notnull"                     json:"volume_liters"`
	CreatedAt    time.Time `bun:"created_at,default:now()"                  json:"created_at"`
}

type WaterInput struct {
	SectorID     string  `json:"sector_id"`
	VolumeLiters float64 `json:"volume_liters"`
}

func (w *WaterInput) Validate() error {
	if strings.TrimSpace(w.SectorID) == "" {
		return errors.New("sector_id is required")
	}
	if w.VolumeLiters <= 0 || w.VolumeLiters > 1000 {
		return errors.New("volume must be between 0 and 1000")
	}
	return nil
}

// telemetry

type Telemetry struct {
	bun.BaseModel `bun:"table:telemetry"`

	ID           string    `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	SectorID     string    `bun:"sector_id,notnull,type:uuid"               json:"sector_id"`
	SoilMoisture float64   `bun:"soil_moisture,notnull"                     json:"soil_moisture"`
	Temperature  float64   `bun:"temperature,notnull"                       json:"temperature"`
	HealthIndex  float64   `bun:"health_index,notnull"                      json:"health_index"`
	RecordedAt   time.Time `bun:"recorded_at,default:now()"                 json:"recorded_at"`
}

// notifications

type Notification struct {
	bun.BaseModel `bun:"table:notifications"`

	ID        string    `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	SectorID  string    `bun:"sector_id,notnull,type:uuid"               json:"sector_id"`
	UserID    *string   `bun:"user_id,type:uuid"                         json:"user_id"`
	Kind      string    `bun:"kind,notnull"                              json:"kind"`
	Message   string    `bun:"message,notnull"                           json:"message"`
	IsRead    bool      `bun:"is_read"                                   json:"is_read"`
	CreatedAt time.Time `bun:"created_at,default:now()"                  json:"created_at"`
}
