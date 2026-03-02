package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/watchwith/watchwith/internal/ws"
	"nhooyr.io/websocket"
)

func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	// Verify room exists
	rm := h.manager.GetRoom(slug)
	if rm == nil {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		username = "Guest"
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:5173", "localhost:3000", "*"},
	})
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}

	clientID := uuid.New().String()
	isHost := h.hub.RoomSize(slug) == 0

	client := ws.NewClient(clientID, username, slug, isHost, h.hub, conn)
	h.hub.Register <- client

	ctx := context.Background()
	go client.WritePump(ctx)
	client.ReadPump(ctx)
}
