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

	// guard against a stale JWT whose user was removed/reseeded out of the DB —
	// otherwise the watering_logs.user_id FK insert fails with an opaque 500.
	if _, err := h.store.GetUser(r.Context(), userID); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "сессия устарела: пользователь не найден, войдите заново",
		})
		return
	}

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

	// equipment failure temporarily disables irrigation (chapter 2.3.6)
	if sector.EquipmentLockedTicks > 0 {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "оборудование неисправно, полив временно недоступен",
		})
		return
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

	// reduce root-zone deficit: treat 1 L as 1 mm for gameplay balance.
	// if input exceeds deficit, excess raises soil moisture above field capacity (waterlogging).
	sector.DeficitDr -= input.VolumeLiters
	if sector.DeficitDr < 0 {
		excess := -sector.DeficitDr
		sector.DeficitDr = 0
		// boost SM above 70% (field capacity); engine will gradually drain it
		sector.SoilMoisture = math.Min(100, 70.0+excess*0.5)
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

// Treat clears a pest infestation on a sector (chapter 2.3.6 — the trainee's
// response to a pest-attack event). Allowed for the assigned operator or agronomist.
func (h *WateringHandler) Treat(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	role, _ := r.Context().Value(middleware.RoleKey).(string)

	id := chi.URLParam(r, "id")
	sector, err := h.store.GetSector(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sector not found"})
		return
	}

	if role == "operator" {
		if sector.OperatorID == nil || *sector.OperatorID != userID {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "you are not assigned to this sector"})
			return
		}
	}

	if !sector.PestActive {
		writeJSON(w, http.StatusOK, map[string]any{"sector": sector, "treated": false})
		return
	}

	sector.PestActive = false
	if sector.Status == "pest" {
		sector.Status = "normal"
	}
	sector.UpdatedAt = time.Now()
	h.store.UpdateSector(r.Context(), sector)
	h.hub.Broadcast("sector:update", sector)

	writeJSON(w, http.StatusOK, map[string]any{"sector": sector, "treated": true})
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
