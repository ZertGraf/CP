package simulation

import (
	"context"
	"log"
	"math"
	"math/rand"
	"time"

	"plantation-api/internal/model"
	"plantation-api/internal/storage"
	"plantation-api/internal/ws"
)

const (
	tickInterval = 10 * time.Second

	alphaDrought = 0.03
	alphaFlood   = 0.02
	alphaHeat    = 0.01
	betaRec      = 0.05
	tBase        = 10.0
	taw          = 50.0
	pDepletion   = 0.5
	thetaSat     = 100.0
	thetaAer     = 15.0

	// markov transition probabilities (wet/dry days)
	pDryToWet = 0.20
	pWetToWet = 0.55
)

// phenophase thresholds following extended BBCH scale for litchi
var phenoPhases = []struct {
	minGDD float64
	code   string
	kcb    float64
}{
	{0, "00", 0.45},
	{150, "01", 0.55},
	{400, "03", 0.70},
	{700, "05", 0.85},
	{1000, "06", 0.95},
	{1500, "07", 1.00},
	{2200, "08", 0.85},
}

type Engine struct {
	store   *storage.Storage
	hub     *ws.Hub
	lastDay int
	rng     *rand.Rand
	wasWet  bool
}

func NewEngine(store *storage.Storage, hub *ws.Hub) *Engine {
	return &Engine{
		store:   store,
		hub:     hub,
		lastDay: time.Now().YearDay(),
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *Engine) Start(ctx context.Context) {
	ticker := time.NewTicker(tickInterval)
	log.Printf("simulation engine started, tick every %s", tickInterval)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				log.Println("simulation engine stopped")
				return
			case <-ticker.C:
				e.tick(ctx)
			}
		}
	}()
}

func (e *Engine) tick(ctx context.Context) {
	today := time.Now().YearDay()
	if today != e.lastDay {
		e.lastDay = today
		if err := e.store.ResetAllWaterConsumed(ctx); err != nil {
			log.Printf("failed to reset water_consumed: %v", err)
		} else {
			log.Println("midnight reset: water_consumed set to 0 for all sectors")
		}
	}

	sectors, err := e.store.ListSectors(ctx)
	if err != nil {
		log.Printf("simulation tick error: %v", err)
		return
	}

	for i := range sectors {
		e.simulateSector(ctx, &sectors[i])
	}
}

func (e *Engine) simulateSector(ctx context.Context, sec *model.Sector) {
	// --- 1. WeatherTick: markov chain for wet/dry state ---
	var pWet float64
	if e.wasWet {
		pWet = pWetToWet
	} else {
		pWet = pDryToWet
	}
	isWetDay := e.rng.Float64() < pWet
	e.wasWet = isWetDay

	tMeanBase := 25.0
	tMax := tMeanBase + 5.0 + (e.rng.Float64()*4 - 2)
	tMin := tMeanBase - 5.0 + (e.rng.Float64()*4 - 2)

	// anomalous heat event (Monte Carlo, ~3% per tick in dry weather)
	if !isWetDay && e.rng.Float64() < 0.03 {
		tMax += 5 + e.rng.Float64()*3
	}

	tMean := (tMax + tMin) / 2.0
	sec.Temperature = tMean

	var precip float64
	if isWetDay {
		// simplified gamma distribution via inverse transform
		precip = -5.0 * math.Log(1.0-e.rng.Float64())
		if precip > 30 {
			precip = 30
		}
	}

	// --- 2. ET0Tick: Hargreaves-Samani equation ---
	ra := 15.0
	et0 := 0.0023 * ra * (tMean + 17.8) * math.Sqrt(math.Max(0, tMax-tMin))

	// dual crop coefficient: ET_c = (Ks * Kcb + Ke) * ET0
	kcb := kcbForPhase(sec.Phenophase)
	ke := 0.10
	etc := (sec.KsWater*kcb + ke) * et0

	// --- 3. WaterBalanceTick ---
	// irrigation I(t) is applied directly by the watering API handler;
	// the engine handles natural processes: precipitation and ET
	peff := precip * 0.8

	netDr := sec.DeficitDr - peff + etc
	if netDr < 0 {
		netDr = 0
	}
	sec.DeficitDr = netDr

	// convert deficit to volumetric moisture
	sec.SoilMoisture = math.Max(0, thetaSat-(sec.DeficitDr/taw)*(thetaSat-20))
	sec.SoilMoisture = math.Min(100, sec.SoilMoisture)

	// Ks water stress coefficient (FAO-56)
	if sec.DeficitDr <= pDepletion*taw {
		sec.KsWater = 1.0
	} else {
		sec.KsWater = math.Max(0, (taw-sec.DeficitDr)/((1.0-pDepletion)*taw))
	}

	// --- 4. PhenologyTick: GDD accumulation ---
	gdd := math.Max(0, tMean-tBase)
	sec.GddCumulative += gdd
	sec.Phenophase = phaseForGDD(sec.GddCumulative)

	// --- 5. StressTick ---
	// aeration stress (waterlogging)
	if sec.SoilMoisture <= thetaSat-thetaAer {
		sec.KsAeration = 1.0
	} else {
		sec.KsAeration = math.Max(0, (thetaSat-sec.SoilMoisture)/thetaAer)
	}

	// heat stress coefficient
	ksT := 1.0
	if tMax > 35.0 {
		ksT = math.Max(0, 1.0-(tMax-35.0)*0.1)
	}

	// --- 6. HealthTick ---
	health := sec.HealthIndex
	deltaDrought := alphaDrought * (1.0 - sec.KsWater)
	deltaFlood := alphaFlood * (1.0 - sec.KsAeration)
	deltaHeat := alphaHeat * (1.0 - ksT)

	rec := 0.0
	if sec.KsWater > 0.9 && sec.KsAeration > 0.9 && ksT > 0.9 {
		rec = betaRec * sec.KsWater * sec.KsAeration * (1.0 - health)
	}

	health = health - deltaDrought - deltaFlood - deltaHeat + rec
	health = math.Max(0, math.Min(1.0, health))
	sec.HealthIndex = health

	// --- 7. EventTick: Monte Carlo random events ---
	// pest attack — probability grows with GDD
	pestProb := 0.02
	if sec.GddCumulative > 700 {
		pestProb = 0.04
	}
	if e.rng.Float64() < pestProb {
		sec.HealthIndex = math.Max(0, sec.HealthIndex-0.05)
	}

	// equipment failure — temporary water limit reduction (~2% per tick)
	if e.rng.Float64() < 0.02 {
		sec.DailyWaterLimit = math.Max(50, sec.DailyWaterLimit*0.6)
	}

	// --- 8. StatusTick: state machine ---
	switch {
	case sec.HealthIndex < 0.1:
		sec.Status = "dead"
	case sec.HealthIndex < 0.3:
		sec.Status = "critical"
	case sec.KsWater < 0.3:
		sec.Status = "drought"
	case sec.KsAeration < 0.5:
		sec.Status = "overwatered"
	case ksT < 0.7:
		sec.Status = "heat_stress"
	case sec.KsWater > 0.85 && sec.KsAeration > 0.85 && sec.HealthIndex < 0.7 && sec.HealthIndex >= 0.3:
		sec.Status = "recovering"
	default:
		sec.Status = "normal"
	}

	sec.UpdatedAt = time.Now()
	e.store.UpdateSector(ctx, sec)

	// save telemetry point
	tele := &model.Telemetry{
		SectorID:     sec.ID,
		SoilMoisture: sec.SoilMoisture,
		Temperature:  sec.Temperature,
		HealthIndex:  sec.HealthIndex,
		RecordedAt:   time.Now(),
	}
	e.store.SaveTelemetry(ctx, tele)

	// broadcast and check alerts
	e.hub.Broadcast("sector:update", sec)
	e.checkAlerts(ctx, sec)
}

func kcbForPhase(code string) float64 {
	for i := len(phenoPhases) - 1; i >= 0; i-- {
		if code >= phenoPhases[i].code {
			return phenoPhases[i].kcb
		}
	}
	return 0.45
}

func phaseForGDD(gdd float64) string {
	result := "00"
	for _, pp := range phenoPhases {
		if gdd >= pp.minGDD {
			result = pp.code
		}
	}
	return result
}

func (e *Engine) checkAlerts(ctx context.Context, sec *model.Sector) {
	var kind, message string

	switch {
	case sec.HealthIndex < 0.1:
		kind = "plant_dead"
		message = "Гибель растений на секторе " + sec.Name + ". Сектор требует пересадки."
	case sec.KsWater < 0.2:
		kind = "critical_drought"
		message = "Критическая засуха на секторе " + sec.Name + ". Требуется немедленный полив."
	case sec.KsWater < 0.5:
		kind = "drought_warning"
		message = "Низкая влажность на секторе " + sec.Name + ". Рекомендуется полив."
	case sec.KsAeration < 0.5:
		kind = "flood_warning"
		message = "Переувлажнение на секторе " + sec.Name + ". Прекратите полив."
	case sec.HealthIndex < 0.3:
		kind = "health_critical"
		message = "Критический индекс здоровья на секторе " + sec.Name + "."
	default:
		return
	}

	notif := &model.Notification{
		SectorID: sec.ID,
		UserID:   sec.OperatorID,
		Kind:     kind,
		Message:  message,
	}
	e.store.CreateNotification(ctx, notif)
	e.hub.Broadcast("notification", notif)
}
