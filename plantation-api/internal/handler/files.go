package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"plantation-api/internal/model"
	"plantation-api/internal/storage"
)

type FileHandler struct {
	store *storage.Storage
}

func NewFileHandler(s *storage.Storage) *FileHandler {
	return &FileHandler{store: s}
}

// export sectors as csv
func (h *FileHandler) ExportSectors(w http.ResponseWriter, r *http.Request) {
	sectors, err := h.store.ListSectors(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=sectors.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"name", "area_sqm", "soil_moisture", "temperature", "health_index", "gdd_cumulative", "phenophase", "ks_water", "ks_aeration", "deficit_dr", "status"})

	for _, s := range sectors {
		writer.Write([]string{
			s.Name,
			fmt.Sprintf("%.2f", s.AreaSqm),
			fmt.Sprintf("%.2f", s.SoilMoisture),
			fmt.Sprintf("%.2f", s.Temperature),
			fmt.Sprintf("%.2f", s.HealthIndex),
			fmt.Sprintf("%.2f", s.GddCumulative),
			s.Phenophase,
			fmt.Sprintf("%.2f", s.KsWater),
			fmt.Sprintf("%.2f", s.KsAeration),
			fmt.Sprintf("%.2f", s.DeficitDr),
			s.Status,
		})
	}
}

// import sectors from csv
func (h *FileHandler) ImportSectors(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20) // 10mb max

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid csv"})
		return
	}

	var created int
	for i, row := range records {
		if i == 0 {
			continue // skip header
		}
		if len(row) < 2 {
			continue
		}

		area, _ := strconv.ParseFloat(strings.TrimSpace(row[1]), 64)

		sector := &model.Sector{
			Name:    strings.TrimSpace(row[0]),
			AreaSqm: area,
		}
		if err := h.store.CreateSector(r.Context(), sector); err == nil {
			created++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"imported": created,
	})
}
