package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"plantation-api/internal/middleware"
	"plantation-api/internal/model"
	"plantation-api/internal/storage"
)

type SectorHandler struct {
	store *storage.Storage
}

func NewSectorHandler(s *storage.Storage) *SectorHandler {
	return &SectorHandler{store: s}
}

func (h *SectorHandler) List(w http.ResponseWriter, r *http.Request) {
	sectors, err := h.store.ListSectors(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sectors)
}

// list only sectors assigned to current operator
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

	sector := &model.Sector{
		Name:    input.Name,
		AreaSqm: input.AreaSqm,
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

// assign operator to sector (agronomist only)
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

	// verify sector exists
	if _, err := h.store.GetSector(r.Context(), sectorID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sector not found"})
		return
	}

	// verify user exists and is an operator
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

// unassign operator from sector (agronomist only)
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
