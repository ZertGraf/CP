package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"plantation-api/internal/storage"
)

type ReportHandler struct {
	store *storage.Storage
}

func NewReportHandler(s *storage.Storage) *ReportHandler {
	return &ReportHandler{store: s}
}

// telemetry history for a sector
func (h *ReportHandler) Telemetry(w http.ResponseWriter, r *http.Request) {
	sectorID := chi.URLParam(r, "sectorId")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 1000 {
		limit = n
	}

	rows, err := h.store.ListTelemetry(r.Context(), sectorID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// aggregated report for a sector
func (h *ReportHandler) Summary(w http.ResponseWriter, r *http.Request) {
	sectorID := chi.URLParam(r, "sectorId")

	avgM, avgT, minH, count, err := h.store.GetTelemetryReport(r.Context(), sectorID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	totalLiters, waterEvents, _ := h.store.GetWateringStats(r.Context(), sectorID)

	writeJSON(w, http.StatusOK, map[string]any{
		"sector_id":          sectorID,
		"telemetry_points":   count,
		"avg_soil_moisture":  avgM,
		"avg_temperature":    avgT,
		"min_health_index":   minH,
		"total_water_liters": totalLiters,
		"watering_events":    waterEvents,
	})
}
