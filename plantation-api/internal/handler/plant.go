package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"plantation-api/internal/model"
	"plantation-api/internal/storage"
)

type PlantHandler struct {
	store *storage.Storage
}

func NewPlantHandler(s *storage.Storage) *PlantHandler {
	return &PlantHandler{store: s}
}

func (h *PlantHandler) List(w http.ResponseWriter, r *http.Request) {
	sectorID := r.URL.Query().Get("sector_id")
	plants, err := h.store.ListPlants(r.Context(), sectorID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, plants)
}

func (h *PlantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input model.PlantInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if err := input.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	plant := &model.Plant{
		SectorID:  input.SectorID,
		Species:   input.Species,
		AgeMonths: input.AgeMonths,
		Health:    100,
	}
	if err := h.store.CreatePlant(r.Context(), plant); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, plant)
}

func (h *PlantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeletePlant(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
