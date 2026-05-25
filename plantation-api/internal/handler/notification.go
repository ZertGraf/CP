package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"plantation-api/internal/middleware"
	"plantation-api/internal/storage"
)

type NotificationHandler struct {
	store *storage.Storage
}

func NewNotificationHandler(s *storage.Storage) *NotificationHandler {
	return &NotificationHandler{store: s}
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	unread := r.URL.Query().Get("unread") == "true"

	notifs, err := h.store.ListNotifications(r.Context(), userID, unread)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, notifs)
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.MarkNotificationRead(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
