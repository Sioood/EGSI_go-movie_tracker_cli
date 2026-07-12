package server

import (
	"net/http"
	"time"

	"github.com/movietracker/movie-tracker/internal/config"
	"github.com/movietracker/movie-tracker/internal/service"
)

type backupHandler struct {
	backups *service.BackupService
}

// GET /api/v1/backup/config
func (h *backupHandler) exportConfig(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())
	cfg, err := h.backups.ExportConfig(r.Context(), claims.UserID)
	if err != nil {
		writeInternalError(w, "backup export config", err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// PUT /api/v1/backup/config
func (h *backupHandler) importConfig(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())

	var body config.Config
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := h.backups.ImportConfig(r.Context(), claims.UserID, body); err != nil {
		writeInternalError(w, "backup import config", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /api/v1/backup/state
func (h *backupHandler) exportState(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())
	state, err := h.backups.ExportState(r.Context(), claims.UserID)
	if err != nil {
		writeInternalError(w, "backup export config", err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// PUT /api/v1/backup/state
func (h *backupHandler) importState(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())

	var body config.State
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := h.backups.ImportState(r.Context(), claims.UserID, body); err != nil {
		writeInternalError(w, "backup import state", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /api/v1/backup
func (h *backupHandler) exportSnapshot(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())
	snapshot, err := h.backups.ExportSnapshot(r.Context(), claims.UserID)
	if err != nil {
		writeInternalError(w, "backup export config", err)
		return
	}
	if snapshot.SyncedAt.IsZero() {
		snapshot.SyncedAt = time.Now().UTC()
	}
	writeJSON(w, http.StatusOK, snapshot)
}

// PUT /api/v1/backup
func (h *backupHandler) importSnapshot(w http.ResponseWriter, r *http.Request) {
	claims, _ := claimsFromContext(r.Context())

	var body service.BackupSnapshot
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}
	if err := h.backups.ImportSnapshot(r.Context(), claims.UserID, body); err != nil {
		writeInternalError(w, "backup import snapshot", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
