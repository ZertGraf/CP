package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"plantation-api/internal/middleware"
	"plantation-api/internal/model"
	"plantation-api/internal/storage"
)

// TrainingHandler serves gamification analytics: the leaderboard, an operator's own
// score, and the agronomist-tunable weather/event configuration (chapter 2.4, 2.2.1).
type TrainingHandler struct {
	store *storage.Storage
}

func NewTrainingHandler(s *storage.Storage) *TrainingHandler {
	return &TrainingHandler{store: s}
}

// Leaderboard ranks operators by total score (agronomist view).
func (h *TrainingHandler) Leaderboard(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.Leaderboard(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if rows == nil {
		rows = []model.LeaderboardEntry{}
	}
	writeJSON(w, http.StatusOK, rows)
}

// MyScore returns the current user's training score and badges (operator dashboard).
func (h *TrainingHandler) MyScore(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	ts, err := h.store.GetTrainingScore(r.Context(), userID)
	if err != nil {
		// no score yet — return a zeroed record rather than 404
		writeJSON(w, http.StatusOK, map[string]any{
			"total_score": 0, "badges": []string{}, "avg_health": 0, "water_efficiency": 0, "tick_count": 0,
		})
		return
	}
	if ts.Badges == nil {
		ts.Badges = []string{}
	}
	writeJSON(w, http.StatusOK, ts)
}

// Award lets the agronomist manually grant/deduct points and badges to an operator
// (chapter 2.4 — mentor-driven reinforcement).
func (h *TrainingHandler) Award(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")

	user, err := h.store.GetUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	if user.Role != "operator" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "points can only be awarded to operators"})
		return
	}

	var in model.AwardInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	ts, err := h.store.AwardScore(r.Context(), userID, in.Points, in.AddBadges, in.RemoveBadges)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if ts.Badges == nil {
		ts.Badges = []string{}
	}
	writeJSON(w, http.StatusOK, ts)
}

// GetWeatherConfig returns the active weather generator profile.
func (h *TrainingHandler) GetWeatherConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.store.GetActiveWeatherConfig(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// UpdateWeatherConfig lets the agronomist tune Markov/gamma parameters and the
// per-event probabilities (chapter 2.2.1).
func (h *TrainingHandler) UpdateWeatherConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.store.GetActiveWeatherConfig(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var in model.WeatherConfigInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	clamp01 := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}
	if in.PDryToWet != nil {
		cfg.PDryToWet = clamp01(*in.PDryToWet)
	}
	if in.PWetToWet != nil {
		cfg.PWetToWet = clamp01(*in.PWetToWet)
	}
	if in.GammaShape != nil && *in.GammaShape > 0 {
		cfg.GammaShape = *in.GammaShape
	}
	if in.GammaScale != nil && *in.GammaScale > 0 {
		cfg.GammaScale = *in.GammaScale
	}
	if in.PHeat != nil {
		cfg.PHeat = clamp01(*in.PHeat)
	}
	if in.PPestBase != nil {
		cfg.PPestBase = clamp01(*in.PPestBase)
	}
	if in.PEquipment != nil {
		cfg.PEquipment = clamp01(*in.PEquipment)
	}
	if in.Latitude != nil && *in.Latitude >= -66 && *in.Latitude <= 66 {
		cfg.Latitude = *in.Latitude
	}
	if in.EtMethod != nil && (*in.EtMethod == "hargreaves" || *in.EtMethod == "penman") {
		cfg.EtMethod = *in.EtMethod
	}

	if err := h.store.UpdateWeatherConfig(r.Context(), cfg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}
