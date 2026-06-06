package handler

import (
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"plantation-api/internal/middleware"
	"plantation-api/internal/model"
	"plantation-api/internal/storage"
	"plantation-api/internal/ws"
)

const (
	ovTaw     = 50.0
	ovThetaFC = 70.0
	ovThetaWP = 20.0
	ovPDep    = 0.5
)

type SectorHandler struct {
	store *storage.Storage
	hub   *ws.Hub
}

func NewSectorHandler(s *storage.Storage, hub *ws.Hub) *SectorHandler {
	return &SectorHandler{store: s, hub: hub}
}

func (h *SectorHandler) List(w http.ResponseWriter, r *http.Request) {
	sectors, err := h.store.ListSectors(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sectors)
}

func (h *SectorHandler) ListMy(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	sectors, err := h.store.ListSectorsByOperator(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sectors)
}

func (h *SectorHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sector, err := h.store.GetSector(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, sector)
}

func (h *SectorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input model.SectorInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if err := input.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// SM=60% → DeficitDr = taw*(thetaFC-SM)/(thetaFC-thetaWP) = 50*(70-60)/50 = 10mm
	sector := &model.Sector{
		Name:            input.Name,
		AreaSqm:         input.AreaSqm,
		SoilMoisture:    60.0,
		Temperature:     25.0,
		HealthIndex:     1.0,
		KsWater:         1.0,
		KsAeration:      1.0,
		DeficitDr:       10.0,
		Phenophase:      "00",
		Status:          "normal",
		DailyWaterLimit: 500,
		WaterConsumed:   0,
	}
	if err := h.store.CreateSector(r.Context(), sector); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, sector)
}

func (h *SectorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sector, err := h.store.GetSector(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	var input model.SectorInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if err := input.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	sector.Name = input.Name
	sector.AreaSqm = input.AreaSqm
	sector.UpdatedAt = time.Now()

	if err := h.store.UpdateSector(r.Context(), sector); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sector)
}

func (h *SectorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeleteSector(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *SectorHandler) Assign(w http.ResponseWriter, r *http.Request) {
	sectorID := chi.URLParam(r, "id")

	var input struct {
		OperatorID string `json:"operator_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if input.OperatorID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "operator_id is required"})
		return
	}

	if _, err := h.store.GetSector(r.Context(), sectorID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sector not found"})
		return
	}

	user, err := h.store.GetUser(r.Context(), input.OperatorID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	if user.Role != "operator" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user is not an operator"})
		return
	}

	if err := h.store.AssignOperator(r.Context(), sectorID, input.OperatorID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":      "assigned",
		"sector_id":   sectorID,
		"operator_id": input.OperatorID,
	})
}

func (h *SectorHandler) Unassign(w http.ResponseWriter, r *http.Request) {
	sectorID := chi.URLParam(r, "id")

	if _, err := h.store.GetSector(r.Context(), sectorID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sector not found"})
		return
	}

	if err := h.store.UnassignOperator(r.Context(), sectorID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "unassigned",
		"sector_id": sectorID,
	})
}

// Override lets agronomist directly set any simulation parameter on a sector.
func (h *SectorHandler) Override(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sector, err := h.store.GetSector(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	var input model.SectorOverrideInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// event presets
	switch input.Event {
	case "drought":
		sector.DeficitDr = ovTaw
		sector.SoilMoisture = ovThetaWP
		sector.KsWater = 0
		sector.KsAeration = 1.0
	case "flood":
		sector.DeficitDr = 0
		sector.SoilMoisture = 100.0 // above FC → engine detects waterlogging
		sector.KsAeration = 0
		sector.KsWater = 1.0
	case "heat":
		sector.Temperature = 42
	case "pest":
		sector.HealthIndex = math.Max(0, sector.HealthIndex-0.3)
		sector.PestActive = true
		sector.Status = "pest"
	case "restore":
		sector.DeficitDr = 10.0 // SM=60% at field-capacity model
		sector.SoilMoisture = 60.0
		sector.HealthIndex = 1.0
		sector.KsWater = 1.0
		sector.KsAeration = 1.0
		sector.Temperature = 25
		sector.WaterConsumed = 0
		sector.PestActive = false
		sector.EquipmentLockedTicks = 0
		sector.Status = "normal"
	}

	// direct field overrides
	if input.Temperature != nil {
		sector.Temperature = *input.Temperature
	}
	if input.HealthIndex != nil {
		sector.HealthIndex = math.Max(0, math.Min(1, *input.HealthIndex))
	}
	if input.GddCumulative != nil {
		sector.GddCumulative = math.Max(0, *input.GddCumulative)
		sector.Phenophase = ovPhaseForGDD(sector.GddCumulative)
	}
	if input.DailyWaterLimit != nil {
		sector.DailyWaterLimit = math.Max(1, *input.DailyWaterLimit)
	}
	if input.WaterConsumed != nil {
		sector.WaterConsumed = math.Max(0, *input.WaterConsumed)
	}
	if input.SoilMoisture != nil {
		sm := math.Max(0, math.Min(100, *input.SoilMoisture))
		sector.SoilMoisture = sm
		// deficit from field-capacity model (clamped to [0, taw])
		sector.DeficitDr = math.Max(0, math.Min(ovTaw, ovTaw*(ovThetaFC-sm)/(ovThetaFC-ovThetaWP)))
	} else if input.DeficitDr != nil {
		sector.DeficitDr = math.Max(0, math.Min(ovTaw, *input.DeficitDr))
		sector.SoilMoisture = math.Max(ovThetaWP, ovThetaFC-(sector.DeficitDr/ovTaw)*(ovThetaFC-ovThetaWP))
	}

	// recalculate stress coefficients
	if sector.DeficitDr <= ovPDep*ovTaw {
		sector.KsWater = 1.0
	} else {
		sector.KsWater = math.Max(0, (ovTaw-sector.DeficitDr)/((1-ovPDep)*ovTaw))
	}
	if sector.SoilMoisture <= ovThetaFC {
		sector.KsAeration = 1.0
	} else {
		excess := sector.SoilMoisture - ovThetaFC
		sector.KsAeration = math.Max(0, 1.0-excess/30.0)
	}

	sector.UpdatedAt = time.Now()
	if err := h.store.UpdateSector(r.Context(), sector); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	h.hub.Broadcast("sector:update", sector)
	writeJSON(w, http.StatusOK, sector)
}

var ovPhenoPhases = []struct {
	minGDD float64
	code   string
}{
	{0, "00"}, {150, "01"}, {400, "03"}, {700, "05"},
	{1000, "06"}, {1500, "07"}, {2200, "08"},
}

func ovPhaseForGDD(gdd float64) string {
	result := "00"
	for _, pp := range ovPhenoPhases {
		if gdd >= pp.minGDD {
			result = pp.code
		}
	}
	return result
}
