package handler

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"plantation-api/internal/model"
	"plantation-api/internal/storage"
	"plantation-api/internal/xlsx"
)

// field-capacity soil model constants (mirror the simulation engine)
const (
	fileTaw     = 50.0
	fileThetaFC = 70.0
	fileThetaWP = 20.0
	filePDep    = 0.5
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

type FileHandler struct {
	store *storage.Storage
}

func NewFileHandler(s *storage.Storage) *FileHandler {
	return &FileHandler{store: s}
}

// export sectors as csv. A UTF-8 BOM and ';' delimiter make Cyrillic names and
// columns render correctly in Russian-locale Excel; the import side auto-detects
// the delimiter, so the file round-trips.
func (h *FileHandler) ExportSectors(w http.ResponseWriter, r *http.Request) {
	sectors, err := h.store.ListSectors(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=sectors.csv")
	w.Write(utf8BOM)

	writer := csv.NewWriter(w)
	writer.Comma = ';'
	defer writer.Flush()

	writer.Write([]string{
		"name", "area_sqm", "soil_moisture", "temperature", "health_index",
		"gdd_cumulative", "phenophase", "ks_water", "ks_aeration", "deficit_dr",
		"daily_water_limit", "status",
	})

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
			fmt.Sprintf("%.2f", s.DailyWaterLimit),
			s.Status,
		})
	}
}

// export telemetry of a sector as xlsx for offline analysis by the agronomist
// (chapter 2.2.5).
func (h *FileHandler) ExportTelemetry(w http.ResponseWriter, r *http.Request) {
	sectorID := chi.URLParam(r, "sectorId")

	sector, err := h.store.GetSector(r.Context(), sectorID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sector not found"})
		return
	}

	rows, err := h.store.ListTelemetry(r.Context(), sectorID, 5000)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// chronological order (ListTelemetry returns newest-first)
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}

	sheet := [][]xlsx.Cell{{
		xlsx.Str("Время"),
		xlsx.Str("Влажность почвы, %"),
		xlsx.Str("Температура, °C"),
		xlsx.Str("Индекс здоровья"),
	}}
	for _, t := range rows {
		sheet = append(sheet, []xlsx.Cell{
			xlsx.Str(t.RecordedAt.Format("2006-01-02 15:04:05")),
			xlsx.Num(t.SoilMoisture),
			xlsx.Num(t.Temperature),
			xlsx.Num(t.HealthIndex),
		})
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=telemetry_%s.xlsx", sectorID))

	if err := xlsx.Write(w, "Телеметрия — "+sector.Name, sheet); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
}

// import sectors from csv. Tolerant of BOM, ';' or ',' delimiters, ragged rows and
// decimal commas. Recognises a header row (any order, subset of columns) and falls
// back to positional name,area_sqm when no header is present. Missing simulation
// fields are filled with sane defaults so imported sectors are healthy and usable
// (rather than zeroed and instantly critical).
func (h *FileHandler) ImportSectors(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot read file"})
		return
	}
	data = bytes.TrimPrefix(data, utf8BOM)
	if len(bytes.TrimSpace(data)) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "empty file"})
		return
	}

	// sniff the delimiter from the first line
	delim := ','
	if nl := bytes.IndexByte(data, '\n'); nl >= 0 {
		first := data[:nl]
		if bytes.Count(first, []byte(";")) > bytes.Count(first, []byte(",")) {
			delim = ';'
		}
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = delim
	reader.FieldsPerRecord = -1 // tolerate ragged rows
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid csv: " + err.Error()})
		return
	}
	if len(records) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"imported": 0, "skipped": 0})
		return
	}

	// build a column-name → index map from the header, or fall back to positional
	col := map[string]int{}
	start := 0
	if rowHasHeader(records[0]) {
		for i, c := range records[0] {
			col[strings.ToLower(strings.TrimSpace(c))] = i
		}
		start = 1
	} else {
		col["name"] = 0
		col["area_sqm"] = 1
	}

	getStr := func(row []string, key string) (string, bool) {
		if idx, ok := col[key]; ok && idx >= 0 && idx < len(row) {
			return strings.TrimSpace(row[idx]), true
		}
		return "", false
	}
	getNum := func(row []string, key string) (float64, bool) {
		if s, ok := getStr(row, key); ok && s != "" {
			s = strings.Replace(s, ",", ".", 1) // tolerate decimal comma
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				return v, true
			}
		}
		return 0, false
	}

	var created, skipped int
	for _, row := range records[start:] {
		name, _ := getStr(row, "name")
		if name == "" {
			skipped++
			continue
		}

		// healthy defaults (mirror SectorHandler.Create)
		sec := &model.Sector{
			Name:            name,
			SoilMoisture:    60.0,
			Temperature:     25.0,
			HealthIndex:     1.0,
			KsWater:         1.0,
			KsAeration:      1.0,
			DeficitDr:       10.0,
			Phenophase:      "00",
			Status:          "normal",
			DailyWaterLimit: 500,
		}

		if v, ok := getNum(row, "area_sqm"); ok {
			sec.AreaSqm = math.Max(0, v)
		}
		if v, ok := getNum(row, "temperature"); ok {
			sec.Temperature = v
		}
		if v, ok := getNum(row, "health_index"); ok {
			sec.HealthIndex = clampf(v, 0, 1)
		}
		if v, ok := getNum(row, "gdd_cumulative"); ok {
			sec.GddCumulative = math.Max(0, v)
			sec.Phenophase = ovPhaseForGDD(sec.GddCumulative)
		}
		if s, ok := getStr(row, "phenophase"); ok && s != "" {
			sec.Phenophase = s
		}
		if v, ok := getNum(row, "daily_water_limit"); ok && v > 0 {
			sec.DailyWaterLimit = v
		}

		// moisture / deficit are linked through the field-capacity model: prefer an
		// explicit soil_moisture, else derive it from an explicit deficit, else keep
		// the default.
		if v, ok := getNum(row, "soil_moisture"); ok {
			sec.SoilMoisture = clampf(v, 0, 100)
			sec.DeficitDr = clampf(fileTaw*(fileThetaFC-sec.SoilMoisture)/(fileThetaFC-fileThetaWP), 0, fileTaw)
		} else if v, ok := getNum(row, "deficit_dr"); ok {
			sec.DeficitDr = clampf(v, 0, fileTaw)
			sec.SoilMoisture = math.Max(fileThetaWP, fileThetaFC-(sec.DeficitDr/fileTaw)*(fileThetaFC-fileThetaWP))
		}

		// recompute stress coefficients so the sector is internally consistent
		if sec.DeficitDr <= filePDep*fileTaw {
			sec.KsWater = 1.0
		} else {
			sec.KsWater = math.Max(0, (fileTaw-sec.DeficitDr)/((1-filePDep)*fileTaw))
		}
		if sec.SoilMoisture <= fileThetaFC {
			sec.KsAeration = 1.0
		} else {
			sec.KsAeration = math.Max(0, 1.0-(sec.SoilMoisture-fileThetaFC)/30.0)
		}

		if err := h.store.CreateSector(r.Context(), sec); err == nil {
			created++
		} else {
			skipped++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"imported": created,
		"skipped":  skipped,
	})
}

// rowHasHeader reports whether a row looks like a header (contains a "name" cell).
func rowHasHeader(row []string) bool {
	for _, c := range row {
		if strings.EqualFold(strings.TrimSpace(c), "name") {
			return true
		}
	}
	return false
}

func clampf(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
