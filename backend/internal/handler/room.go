package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/watchwith/watchwith/internal/room"
	"github.com/watchwith/watchwith/internal/ws"
)

type Handler struct {
	manager *room.Manager
	hub     *ws.Hub
}

func New(manager *room.Manager, hub *ws.Hub) *Handler {
	return &Handler{manager: manager, hub: hub}
}

type CreateRoomRequest struct {
	Name string `json:"name"`
}

type CreateRoomResponse struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type RoomResponse struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	PeerCount int    `json:"peerCount"`
}

func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = "Watch Party"
	}

	rm := h.manager.CreateRoom(req.Name)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateRoomResponse{
		Slug: rm.Slug,
		Name: rm.Name,
	})
}

func (h *Handler) GetRoom(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	rm := h.manager.GetRoom(slug)
	if rm == nil {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RoomResponse{
		Slug:      rm.Slug,
		Name:      rm.Name,
		PeerCount: h.hub.RoomSize(slug),
	})
}
