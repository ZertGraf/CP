package storage

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"

	"plantation-api/internal/model"
)

// seed data constants
var sectorNames = []string{
	"Северный склон",
	"Южная терраса",
	"Речная долина",
	"Восточный гребень",
	"Центральная поляна",
	"Западный каньон",
}

var speciesList = []string{
	"Litchi chinensis",
	"Litchi chinensis var. philippinensis",
}

func (s *Storage) Seed(ctx context.Context) error {
	// check if users already exist
	count, err := s.DB.NewSelect().Model((*model.User)(nil)).Count(ctx)
	if err != nil {
		return fmt.Errorf("seed check failed: %w", err)
	}
	if count > 0 {
		log.Println("seed: database already populated, skipping")
		return nil
	}

	log.Println("seed: empty database detected, generating test data...")

	// create users
	agronomist, err := s.seedUser(ctx, "agro@test.com", "123456", "Иванов Пётр Алексеевич", "agronomist")
	if err != nil {
		return fmt.Errorf("seed agronomist: %w", err)
	}
	log.Printf("seed: created agronomist %s (%s)", agronomist.Name, agronomist.Email)

	operators := make([]*model.User, 3)
	opNames := []string{
		"Сидоров Алексей Викторович",
		"Гарсия Мария Хосефина",
		"Чен Вэйминь",
	}
	for i := 0; i < 3; i++ {
		email := fmt.Sprintf("op%d@test.com", i+1)
		operators[i], err = s.seedUser(ctx, email, "123456", opNames[i], "operator")
		if err != nil {
			return fmt.Errorf("seed operator %d: %w", i, err)
		}
		log.Printf("seed: created operator %s (%s)", operators[i].Name, operators[i].Email)
	}

	// create sectors and assign operators
	sectors := make([]*model.Sector, len(sectorNames))
	for i, name := range sectorNames {
		area := 500 + rand.Float64()*2000
		moisture := 30 + rand.Float64()*40
		temp := 24 + rand.Float64()*6
		health := 0.6 + rand.Float64()*0.4

		opID := operators[i%len(operators)].ID

		now := time.Now().Add(-time.Duration(rand.Intn(120)) * time.Minute)

		sector := &model.Sector{
			Name:            name,
			AreaSqm:         float64(int(area*100)) / 100,
			SoilMoisture:    float64(int(moisture*10)) / 10,
			Temperature:     float64(int(temp*10)) / 10,
			HealthIndex:     health,
			GddCumulative:   rand.Float64() * 1000,
			Phenophase:      "00",
			KsWater:         1.0,
			KsAeration:      1.0,
			DeficitDr:       0,
			Status:          "normal",
			OperatorID:      &opID,
			LastWateredAt:   &now,
			DailyWaterLimit: 300 + float64(rand.Intn(400)),
			WaterConsumed:   float64(rand.Intn(150)),
		}

		if sector.SoilMoisture < 20 {
			sector.Status = "drought"
		} else if sector.SoilMoisture > 90 {
			sector.Status = "overwatered"
		}

		if err := s.CreateSector(ctx, sector); err != nil {
			return fmt.Errorf("seed sector %s: %w", name, err)
		}
		sectors[i] = sector
		log.Printf("seed: created sector %q (%.0f m², operator: %s)", sector.Name, sector.AreaSqm, operators[i%len(operators)].Name)
	}

	// create plants for each sector
	plantCount := 0
	for _, sector := range sectors {
		n := 3 + rand.Intn(5)
		for j := 0; j < n; j++ {
			plant := &model.Plant{
				SectorID:  sector.ID,
				Species:   speciesList[rand.Intn(len(speciesList))],
				AgeMonths: 6 + rand.Intn(60),
				Health:    50 + rand.Intn(50),
			}
			if err := s.CreatePlant(ctx, plant); err != nil {
				return fmt.Errorf("seed plant: %w", err)
			}
			plantCount++
		}
	}
	log.Printf("seed: created %d plants across %d sectors", plantCount, len(sectors))

	// generate watering history (last 7 days)
	waterCount := 0
	for day := 6; day >= 0; day-- {
		for _, sector := range sectors {
			events := 2 + rand.Intn(4)
			for e := 0; e < events; e++ {
				opIdx := rand.Intn(len(operators))
				vol := 5 + rand.Float64()*30
				t := time.Now().AddDate(0, 0, -day).Add(
					time.Duration(6+rand.Intn(12)) * time.Hour,
				).Add(
					time.Duration(rand.Intn(60)) * time.Minute,
				)

				wl := &model.WateringLog{
					SectorID:     sector.ID,
					UserID:       operators[opIdx].ID,
					VolumeLiters: float64(int(vol*10)) / 10,
				}
				_, err := s.DB.NewInsert().Model(wl).Exec(ctx)
				if err != nil {
					return fmt.Errorf("seed watering log: %w", err)
				}
				// backdate the created_at
				_, err = s.DB.NewUpdate().Model(wl).
					Set("created_at = ?", t).
					WherePK().Exec(ctx)
				if err != nil {
					return fmt.Errorf("seed watering log date: %w", err)
				}
				waterCount++
			}
		}
	}
	log.Printf("seed: created %d watering log entries (7 days)", waterCount)

	// generate telemetry history (last 24h, every ~10 min)
	telemCount := 0
	for _, sector := range sectors {
		moisture := sector.SoilMoisture
		temp := sector.Temperature
		health := sector.HealthIndex

		// 144 ticks = 24h at 10min intervals
		for tick := 143; tick >= 0; tick-- {
			t := time.Now().Add(-time.Duration(tick) * 10 * time.Minute)

			// simulate small random drift
			moisture += (rand.Float64()*3 - 1.8)
			if moisture < 5 {
				moisture = 5
			}
			if moisture > 95 {
				moisture = 95
			}

			temp += (rand.Float64()*1 - 0.5)
			if temp < 20 {
				temp = 20
			}
			if temp > 38 {
				temp = 38
			}

			if moisture < 20 {
				health -= rand.Float64() * 0.03
			} else if moisture > 85 {
				health -= rand.Float64() * 0.02
			} else {
				health += rand.Float64() * 0.02
			}
			if health > 1.0 {
				health = 1.0
			}
			if health < 0.1 {
				health = 0.1
			}

			tele := &model.Telemetry{
				SectorID:     sector.ID,
				SoilMoisture: float64(int(moisture*10)) / 10,
				Temperature:  float64(int(temp*10)) / 10,
				HealthIndex:  health,
			}
			_, err := s.DB.NewInsert().Model(tele).Exec(ctx)
			if err != nil {
				return fmt.Errorf("seed telemetry: %w", err)
			}
			// backdate
			_, err = s.DB.NewUpdate().Model(tele).
				Set("recorded_at = ?", t).
				WherePK().Exec(ctx)
			if err != nil {
				return fmt.Errorf("seed telemetry date: %w", err)
			}
			telemCount++
		}
	}
	log.Printf("seed: created %d telemetry records (24h history)", telemCount)

	// generate some notifications
	notifCount := 0
	kinds := []struct {
		kind string
		msg  string
	}{
		{"drought_warning", "Низкая влажность на секторе %s. Рекомендуется полив."},
		{"critical_drought", "Критическая засуха на секторе %s. Требуется немедленный полив."},
		{"health_critical", "Критический индекс здоровья на секторе %s."},
		{"flood_warning", "Переувлажнение на секторе %s. Прекратите полив."},
	}
	for _, sector := range sectors {
		n := 1 + rand.Intn(3)
		for j := 0; j < n; j++ {
			k := kinds[rand.Intn(len(kinds))]
			t := time.Now().Add(-time.Duration(rand.Intn(24)) * time.Hour)
			notif := &model.Notification{
				SectorID: sector.ID,
				UserID:   sector.OperatorID,
				Kind:     k.kind,
				Message:  fmt.Sprintf(k.msg, sector.Name),
				IsRead:   rand.Float64() < 0.3,
			}
			_, err := s.DB.NewInsert().Model(notif).Exec(ctx)
			if err != nil {
				return fmt.Errorf("seed notification: %w", err)
			}
			_, _ = s.DB.NewUpdate().Model(notif).
				Set("created_at = ?", t).
				WherePK().Exec(ctx)
			notifCount++
		}
	}
	log.Printf("seed: created %d notifications", notifCount)

	log.Println("seed: done! test accounts:")
	log.Println("  agro@test.com / 123456 (agronomist)")
	log.Println("  op1@test.com  / 123456 (operator)")
	log.Println("  op2@test.com  / 123456 (operator)")
	log.Println("  op3@test.com  / 123456 (operator)")

	return nil
}

func (s *Storage) seedUser(ctx context.Context, email, password, name, role string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &model.User{
		Email:    email,
		Password: string(hash),
		Name:     name,
		Role:     role,
	}
	if err := s.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
