package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"plantation-api/internal/middleware"
	"plantation-api/internal/model"
	"plantation-api/internal/storage"
	"plantation-api/internal/ws"
)

type WateringHandler struct {
	store *storage.Storage
	hub   *ws.Hub
}

func NewWateringHandler(s *storage.Storage, hub *ws.Hub) *WateringHandler {
	return &WateringHandler{store: s, hub: hub}
}

func (h *WateringHandler) Water(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	role, _ := r.Context().Value(middleware.RoleKey).(string)

	var input model.WaterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if err := input.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	sector, err := h.store.GetSector(r.Context(), input.SectorID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sector not found"})
		return
	}

	// operators can only water their assigned sectors
	if role == "operator" {
		if sector.OperatorID == nil || *sector.OperatorID != userID {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error": "you are not assigned to this sector",
			})
			return
		}
	}

	// check daily limit
	if sector.WaterConsumed+input.VolumeLiters > sector.DailyWaterLimit {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "daily water limit exceeded",
		})
		return
	}

	// save watering log
	logEntry := &model.WateringLog{
		SectorID:     input.SectorID,
		UserID:       userID,
		VolumeLiters: input.VolumeLiters,
	}
	if err := h.store.CreateWateringLog(r.Context(), logEntry); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// update sector state
	// Convert volume directly to mm (assuming 1L / m^2 = 1mm). 
	// For simplicity, let's say volume reduces deficit directly.
	if sector.AreaSqm > 0 {
		mmAdded := input.VolumeLiters / sector.AreaSqm
		sector.DeficitDr -= mmAdded
		if sector.DeficitDr < 0 {
			sector.DeficitDr = 0
		}
	}

	sector.WaterConsumed += input.VolumeLiters
	now := time.Now()
	sector.LastWateredAt = &now
	sector.UpdatedAt = now

	h.store.UpdateSector(r.Context(), sector)

	// broadcast update
	h.hub.Broadcast("sector:watered", map[string]any{
		"sector":  sector,
		"watered": logEntry,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"log":    logEntry,
		"sector": sector,
	})
}

func (h *WateringHandler) Stats(w http.ResponseWriter, r *http.Request) {
	sectorID := chi.URLParam(r, "sectorId")
	total, count, err := h.store.GetWateringStats(r.Context(), sectorID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sector_id":    sectorID,
		"total_liters": total,
		"total_events": count,
	})
}
