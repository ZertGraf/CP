package simulation

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"plantation-api/internal/model"
	"plantation-api/internal/storage"
	"plantation-api/internal/ws"
)

const (
	tickInterval = 10 * time.Second

	// tickScale compresses daily ET and precipitation into 10-second ticks.
	// 0.05 ≈ 1 simulated day per real minute; soil lasts ~20 min before needing water.
	tickScale = 0.05

	alphaDrought = 0.03
	alphaFlood   = 0.02
	alphaHeat    = 0.01
	alphaPest    = 0.03 // ongoing health drain while an infestation is untreated
	betaRec      = 0.05
	tBase        = 10.0
	taw          = 50.0
	pDepletion   = 0.5

	// Field-capacity model: DeficitDr=0 → soil at 70% (not 100% saturated).
	// Saturation (>70%) only from explicit flood events / heavy over-watering.
	thetaFC  = 70.0
	thetaWP  = 20.0
	thetaAer = 30.0 // mm of excess above FC for full aeration stress

	// AR(1) air-temperature generator parameters (chapter 2.3.1).
	arRho   = 0.70 // autoregression coefficient
	arSigma = 2.5  // innovation std-dev, °C

	// gamification badge thresholds (simulated days = ticks)
	badgeWaterKeeperDays  = 7
	badgeGreenMasterDays  = 14
	badgeCrisisManagerRun = 3
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

// weatherState is the regional weather generated once per tick and shared across
// all sectors (one plantation, one climate). Subsystem 1 of the tick pipeline.
type weatherState struct {
	isWet     bool
	tMax      float64
	tMin      float64
	tMean     float64
	precip    float64 // daily depth, mm
	heatEvent bool
}

type Engine struct {
	store     *storage.Storage
	hub       *ws.Hub
	lastDay   int
	rng       *rand.Rand
	baseSeed  int64
	wasWet    bool
	tempResid float64 // AR(1) residual carried between ticks
	tickCount int64
	sessionID string
}

func NewEngine(store *storage.Storage, hub *ws.Hub) *Engine {
	// SIM_SEED makes the whole simulation deterministic and replayable for the
	// agronomist's action analytics (chapter 2.5); unset → fresh random seed.
	seed := time.Now().UnixNano()
	if v := os.Getenv("SIM_SEED"); v != "" {
		if s, err := strconv.ParseInt(v, 10, 64); err == nil {
			seed = s
			log.Printf("simulation: deterministic mode, seed=%d", seed)
		}
	}
	return &Engine{
		store:     store,
		hub:       hub,
		lastDay:   time.Now().YearDay(),
		rng:       rand.New(rand.NewSource(seed)),
		baseSeed:  seed,
		sessionID: genUUID(),
	}
}

func (e *Engine) Start(ctx context.Context) {
	ticker := time.NewTicker(tickInterval)
	log.Printf("simulation engine started, tick every %s, session=%s", tickInterval, e.sessionID)

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

	cfg, err := e.store.GetActiveWeatherConfig(ctx)
	if err != nil {
		log.Printf("weather config load failed, using default: %v", err)
		cfg = model.DefaultWeatherConfig()
	}

	// --- subsystem 1: WeatherTick (generated once, regionally shared) ---
	wx := e.generateWeather(cfg)
	e.tickCount++

	sectors, err := e.store.ListSectors(ctx)
	if err != nil {
		log.Printf("simulation tick error: %v", err)
		return
	}

	// each sector evolves independently → fan out across goroutines (chapter 2.5).
	results := make([]*scoreResult, len(sectors))
	var wg sync.WaitGroup
	for i := range sectors {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// per-sector RNG keeps the run deterministic yet decorrelated between sectors.
			rng := rand.New(rand.NewSource(e.baseSeed + e.tickCount*1_000_003 + int64(idx)))
			results[idx] = e.simulateSector(ctx, &sectors[idx], wx, cfg, rng)
		}(i)
	}
	wg.Wait()

	// --- subsystem 8: ScoringTick (applied sequentially to avoid row races) ---
	for _, res := range results {
		if res == nil || res.operatorID == "" {
			continue
		}
		if err := e.store.ApplyScore(ctx, res.operatorID, e.sessionID,
			res.delta, res.health, res.efficiency, res.badges); err != nil {
			log.Printf("scoring failed for %s: %v", res.operatorID, err)
		}
	}
}

// generateWeather runs the Markov wet/dry chain, the AR(1) temperature model and the
// gamma rainfall sampler (chapter 2.3.1, equations 2.1).
func (e *Engine) generateWeather(cfg *model.WeatherConfig) weatherState {
	pWet := cfg.PDryToWet
	if e.wasWet {
		pWet = cfg.PWetToWet
	}
	isWet := e.rng.Float64() < pWet
	e.wasWet = isWet

	// AR(1): resid_t = ρ·resid_{t-1} + σ·sqrt(1-ρ²)·z, conditioned on wet/dry state.
	z := e.rng.NormFloat64()
	e.tempResid = arRho*e.tempResid + arSigma*math.Sqrt(1.0-arRho*arRho)*z

	seasonalMean := 25.0
	wetOffset := 0.0
	amplitude := 6.0
	if isWet {
		wetOffset = -2.5 // overcast wet days are cooler
		amplitude = 4.5  // smaller diurnal range under cloud cover
	}
	tMean := seasonalMean + wetOffset + e.tempResid
	tMax := tMean + amplitude
	tMin := tMean - amplitude

	wx := weatherState{isWet: isWet}

	// anomalous heat event (Monte Carlo) — only in dry weather.
	if !isWet && e.rng.Float64() < cfg.PHeat {
		bump := 5.0 + e.rng.Float64()*3.0
		tMax += bump
		tMean += bump / 2.0
		wx.heatEvent = true
	}

	if isWet {
		precip := sampleGamma(e.rng, cfg.GammaShape, cfg.GammaScale)
		if precip > 60 {
			precip = 60
		}
		wx.precip = precip
	}

	wx.tMax = tMax
	wx.tMin = tMin
	wx.tMean = (tMax + tMin) / 2.0
	return wx
}

type scoreResult struct {
	operatorID string
	delta      float64
	health     float64
	efficiency float64
	badges     []string
}

func (e *Engine) simulateSector(ctx context.Context, sec *model.Sector, wx weatherState, cfg *model.WeatherConfig, rng *rand.Rand) *scoreResult {
	// equipment lock decays at the start of the tick (chapter 2.3.6 — failure limits
	// available watering for 1–3 ticks rather than directly damaging the plant).
	if sec.EquipmentLockedTicks > 0 {
		sec.EquipmentLockedTicks--
	}

	sec.Temperature = wx.tMean

	// --- 2. ET0Tick: Hargreaves-Samani or full Penman-Monteith (FAO-56) ---
	doy := time.Now().YearDay()
	var et0 float64
	if cfg.EtMethod == "penman" {
		et0 = et0PenmanMonteith(wx.tMax, wx.tMin, cfg.Latitude, doy, 2.0) * tickScale
	} else {
		et0 = et0Hargreaves(wx.tMax, wx.tMin, cfg.Latitude, doy) * tickScale
	}

	// dual crop coefficient: ET_c = (Ks * Kcb + Ke) * ET0
	kcb := kcbForPhase(sec.Phenophase)
	ke := 0.10
	etc := (sec.KsWater*kcb + ke) * et0

	// --- 3. WaterBalanceTick ---
	peff := wx.precip * 0.8 * tickScale

	netDr := sec.DeficitDr - peff + etc
	if netDr < 0 {
		netDr = 0
	}
	sec.DeficitDr = math.Min(taw, netDr)

	// field-capacity soil moisture: DeficitDr=0 → 70% (not 100% saturated).
	normalSM := math.Max(thetaWP, thetaFC-(sec.DeficitDr/taw)*(thetaFC-thetaWP))
	if sec.SoilMoisture > thetaFC {
		sec.SoilMoisture = math.Max(normalSM, sec.SoilMoisture-10.0*tickScale)
	} else {
		sec.SoilMoisture = normalSM
	}

	// Ks water stress coefficient (FAO-56)
	if sec.DeficitDr <= pDepletion*taw {
		sec.KsWater = 1.0
	} else {
		sec.KsWater = math.Max(0, (taw-sec.DeficitDr)/((1.0-pDepletion)*taw))
	}

	// --- 4. PhenologyTick: GDD accumulation ---
	// scaled by tickScale to match the compressed-day model used for ET and rainfall,
	// so phenophases progress through the BBCH scale gradually instead of saturating.
	gdd := math.Max(0, wx.tMean-tBase) * tickScale
	sec.GddCumulative += gdd
	sec.Phenophase = phaseForGDD(sec.GddCumulative)

	// --- 5. StressTick ---
	if sec.SoilMoisture <= thetaFC {
		sec.KsAeration = 1.0
	} else {
		excess := sec.SoilMoisture - thetaFC
		sec.KsAeration = math.Max(0, 1.0-excess/thetaAer)
	}

	ksT := 1.0
	if wx.tMax > 35.0 {
		ksT = math.Max(0, 1.0-(wx.tMax-35.0)*0.1)
	}

	// CWSI for the agronomist dashboard (chapter 2.3.5, eq. 2.10)
	sec.Cwsi = cropWaterStressIndex(sec.KsWater, wx.tMean)

	// --- 6. HealthTick ---
	health := sec.HealthIndex
	deltaDrought := alphaDrought * (1.0 - sec.KsWater)
	deltaFlood := alphaFlood * (1.0 - sec.KsAeration)
	deltaHeat := alphaHeat * (1.0 - ksT)
	deltaPest := 0.0
	if sec.PestActive {
		deltaPest = alphaPest // sustained damage until the trainee treats the sector
	}

	rec := 0.0
	if sec.KsWater > 0.9 && sec.KsAeration > 0.9 && ksT > 0.9 && !sec.PestActive {
		rec = betaRec * sec.KsWater * sec.KsAeration * (1.0 - health)
	}

	health = health - deltaDrought - deltaFlood - deltaHeat - deltaPest + rec
	health = math.Max(0, math.Min(1.0, health))
	sec.HealthIndex = health

	// --- 7. EventTick: Monte Carlo random events ---
	eventFired := wx.heatEvent || sec.KsAeration < 0.7 || sec.PestActive

	// pest attack — probability grows with accumulated GDD; starts a persistent
	// infestation that keeps draining health until the operator applies treatment.
	if !sec.PestActive {
		pestProb := cfg.PPestBase
		if sec.GddCumulative > 700 {
			pestProb *= 2.0
		}
		if rng.Float64() < pestProb {
			sec.PestActive = true
			sec.HealthIndex = math.Max(0, sec.HealthIndex-0.05) // initial bite
			eventFired = true
			e.notifyPest(ctx, sec)
		}
	}

	// equipment failure — restricts watering for the next 1–3 ticks
	if sec.EquipmentLockedTicks == 0 && rng.Float64() < cfg.PEquipment {
		sec.EquipmentLockedTicks = 1 + rng.Intn(3)
		e.notifyEquipment(ctx, sec)
		eventFired = true
	}

	// --- 8. StatusTick: state machine ---
	switch {
	case sec.HealthIndex < 0.1:
		sec.Status = "dead"
	case sec.HealthIndex < 0.3:
		sec.Status = "critical"
	case sec.PestActive:
		sec.Status = "pest"
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

	// --- scoring & badges (per sector, attributed to its operator) ---
	res := e.scoreSector(sec, eventFired)

	// alerts: emit only when the alert state changes, not every tick (no spam)
	kind, message := alertFor(sec)
	newAlert := kind != "" && kind != sec.LastAlertKind
	sec.LastAlertKind = kind

	sec.UpdatedAt = time.Now()
	e.store.UpdateSector(ctx, sec)

	tele := &model.Telemetry{
		SectorID:     sec.ID,
		SoilMoisture: sec.SoilMoisture,
		Temperature:  sec.Temperature,
		HealthIndex:  sec.HealthIndex,
		RecordedAt:   time.Now(),
	}
	e.store.SaveTelemetry(ctx, tele)

	e.hub.Broadcast("sector:update", sec)

	if newAlert {
		notif := &model.Notification{
			SectorID: sec.ID,
			UserID:   sec.OperatorID,
			Kind:     kind,
			Message:  message,
		}
		e.store.CreateNotification(ctx, notif)
		e.hub.Broadcast("notification", notif)
	}
	return res
}

// scoreSector computes the per-tick score contribution and updates the streak
// counters that drive badge awards (chapter 2.4.1–2.4.2).
func (e *Engine) scoreSector(sec *model.Sector, eventFired bool) *scoreResult {
	if sec.OperatorID == nil {
		return nil
	}

	delta := 0.0
	if sec.HealthIndex > 0.7 {
		delta += 1.0
	}
	if sec.KsWater > 0.9 && sec.KsAeration > 0.9 {
		delta += 0.5 // efficient, no-stress watering
	}
	if sec.KsWater < 0.3 {
		delta -= 2.0 // allowed drought
	}
	if sec.KsAeration < 0.5 {
		delta -= 2.0 // allowed waterlogging
	}

	var badges []string

	// "Хранитель воды" — 7 simulated days without waterlogging
	if sec.KsAeration > 0.9 {
		sec.SafeStreak++
		if sec.SafeStreak == badgeWaterKeeperDays {
			badges = append(badges, "water_keeper")
			delta += 5.0
		}
	} else if sec.KsAeration < 0.5 {
		sec.SafeStreak = 0
	}

	// "Зелёный мастер" — H > 0.9 for 14 simulated days
	if sec.HealthIndex > 0.9 {
		sec.HealthyStreak++
		if sec.HealthyStreak == badgeGreenMasterDays {
			badges = append(badges, "green_master")
			delta += 5.0
		}
	} else {
		sec.HealthyStreak = 0
	}

	// "Кризис-менеджер" — survive 3 random events in a row keeping H healthy
	if sec.HealthIndex < 0.3 {
		sec.CrisisStreak = 0
	} else if eventFired && sec.HealthIndex > 0.5 {
		sec.CrisisStreak++
		if sec.CrisisStreak == badgeCrisisManagerRun {
			badges = append(badges, "crisis_manager")
			delta += 5.0
		}
	}

	return &scoreResult{
		operatorID: *sec.OperatorID,
		delta:      delta,
		health:     sec.HealthIndex,
		efficiency: sec.KsWater * sec.KsAeration,
		badges:     badges,
	}
}

func (e *Engine) notifyPest(ctx context.Context, sec *model.Sector) {
	notif := &model.Notification{
		SectorID: sec.ID,
		UserID:   sec.OperatorID,
		Kind:     "pest_attack",
		Message:  "Нашествие вредителей на секторе " + sec.Name + ". Требуется обработка.",
	}
	e.store.CreateNotification(ctx, notif)
	e.hub.Broadcast("notification", notif)
}

func (e *Engine) notifyEquipment(ctx context.Context, sec *model.Sector) {
	notif := &model.Notification{
		SectorID: sec.ID,
		UserID:   sec.OperatorID,
		Kind:     "equipment_failure",
		Message:  "Поломка оборудования на секторе " + sec.Name + ". Полив недоступен на несколько тиков.",
	}
	e.store.CreateNotification(ctx, notif)
	e.hub.Broadcast("notification", notif)
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

// alertFor returns the alert kind/message for a sector's current state, or empty
// strings when the sector is fine. The caller emits a notification only when this
// transitions to a new kind (see simulateSector), preventing per-tick spam.
func alertFor(sec *model.Sector) (kind, message string) {
	switch {
	case sec.HealthIndex < 0.1:
		return "plant_dead", "Гибель растений на секторе " + sec.Name + ". Сектор требует пересадки."
	case sec.KsWater < 0.2:
		return "critical_drought", "Критическая засуха на секторе " + sec.Name + ". Требуется немедленный полив."
	case sec.KsWater < 0.5:
		return "drought_warning", "Низкая влажность на секторе " + sec.Name + ". Рекомендуется полив."
	case sec.KsAeration < 0.5:
		return "flood_warning", "Переувлажнение на секторе " + sec.Name + ". Прекратите полив."
	case sec.HealthIndex < 0.3:
		return "health_critical", "Критический индекс здоровья на секторе " + sec.Name + "."
	}
	return "", ""
}

// genUUID returns a random RFC-4122 v4 UUID string for the training session.
func genUUID() string {
	b := make([]byte, 16)
	if _, err := crand.Read(b); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
