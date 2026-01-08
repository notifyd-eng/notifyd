package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/notifyd-eng/notifyd/internal/store"
)

type createRequest struct {
	Channel   string `json:"channel"`
	Recipient string `json:"recipient"`
	Subject   string `json:"subject,omitempty"`
	Body      string `json:"body"`
	Priority  int    `json:"priority,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if req.Channel == "" || req.Recipient == "" || req.Body == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "channel, recipient, and body are required"})
		return
	}

	validChannels := map[string]bool{"email": true, "slack": true, "webhook": true, "sms": true}
	if !validChannels[req.Channel] {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid channel: must be email, slack, webhook, or sms"})
		return
	}

	id, err := generateID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to generate notification ID"})
		return
	}

	n := &store.Notification{
		ID:        id,
		Channel:   req.Channel,
		Recipient: req.Recipient,
		Subject:   req.Subject,
		Body:      req.Body,
		Priority:  req.Priority,
	}

	if err := s.store.Insert(n); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create notification"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": id, "status": "pending"})
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := s.store.Get(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		return
	}
	if n == nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "notification not found"})
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	f := store.ListFilter{
		Channel: r.URL.Query().Get("channel"),
		Status:  r.URL.Query().Get("status"),
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			f.Limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			f.Offset = v
		}
	}

	if f.Limit == 0 || f.Limit > 100 {
		f.Limit = 50
	}

	results, err := s.store.List(f)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"notifications": results,
		"count":         len(results),
	})
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := s.store.Get(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		return
	}
	if n == nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "notification not found"})
		return
	}
	if n.Status != "pending" {
		writeJSON(w, http.StatusConflict, errorResponse{Error: "can only cancel pending notifications"})
		return
	}

	if err := s.store.UpdateStatus(id, "cancelled"); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to cancel"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.Stats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "ntf_" + hex.EncodeToString(b), nil
}
