package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"korp/faturamento/internal/repository"
)

type LogsHandler struct {
	Repo *repository.ActivityRepository
}

func (h *LogsHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	logs, err := h.Repo.List(r.Context(), limit)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func (h *LogsHandler) Log(ctx context.Context, event, entity string, details interface{}) {
	if err := h.Repo.Log(ctx, event, entity, details); err != nil {
		log.Printf("error writing activity log (%s): %v", event, err)
	}
}