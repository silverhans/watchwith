package room

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/watchwith/watchwith/internal/store"
	"github.com/watchwith/watchwith/internal/ws"
)

const slugChars = "abcdefghijklmnopqrstuvwxyz0123456789"
const slugLength = 8
const chatHistorySize = 100

type Manager struct {
	rooms       map[string]*Room // keyed by slug
	chatHistory map[string][]ws.ChatPayload
	mu          sync.RWMutex
	hub         *ws.Hub
	db          *store.DB // nil = in-memory only
}

func NewManager(hub *ws.Hub, db *store.DB) *Manager {
	m := &Manager{
		rooms:       make(map[string]*Room),
		chatHistory: make(map[string][]ws.ChatPayload),
		hub:         hub,
		db:          db,
	}
	m.registerHandlers()
	return m
}

func (m *Manager) registerHandlers() {
	m.hub.OnMessage(ws.TypeRoomJoin, m.handleJoin)
	m.hub.OnMessage(ws.TypeRoomLeave, m.handleLeave)
	m.hub.OnMessage(ws.TypePlayerPlay, m.handlePlayerPlay)
	m.hub.OnMessage(ws.TypePlayerPause, m.handlePlayerPause)
	m.hub.OnMessage(ws.TypePlayerSeek, m.handlePlayerSeek)
	m.hub.OnMessage(ws.TypePlayerSource, m.handlePlayerSource)
	m.hub.OnMessage(ws.TypePlayerSync, m.handlePlayerSync)
	m.hub.OnMessage(ws.TypeChatMessage, m.handleChat)
	m.hub.OnMessage(ws.TypeRTCOffer, m.handleRTCRelay)
	m.hub.OnMessage(ws.TypeRTCAnswer, m.handleRTCRelay)
	m.hub.OnMessage(ws.TypeRTCICE, m.handleRTCRelay)
	m.hub.OnMessage(ws.TypeScreenStart, m.handleScreenShare)
	m.hub.OnMessage(ws.TypeScreenStop, m.handleScreenShare)
}

func (m *Manager) CreateRoom(name string) *Room {
	id := uuid.New().String()
	slug := generateSlug()

	room := NewRoom(id, slug, name, "")

	// Persist to DB
	if m.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := m.db.CreateRoom(ctx, id, slug, name); err != nil {
			log.Printf("failed to persist room to DB: %v", err)
		}
	}

	m.mu.Lock()
	m.rooms[slug] = room
	m.chatHistory[slug] = make([]ws.ChatPayload, 0, chatHistorySize)
	m.mu.Unlock()

	log.Printf("room created: slug=%s name=%s", slug, name)
	return room
}

func (m *Manager) GetRoom(slug string) *Room {
	m.mu.RLock()
	room := m.rooms[slug]
	m.mu.RUnlock()
	if room != nil {
		return room
	}

	// Try loading from DB
	if m.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		row, err := m.db.GetRoomBySlug(ctx, slug)
		if err != nil {
			log.Printf("failed to load room from DB: %v", err)
			return nil
		}
		if row == nil {
			return nil
		}

		room = NewRoom(row.ID, row.Slug, row.Name, "")
		if row.VideoURL != "" {
			room.Video.URL = row.VideoURL
		}

		m.mu.Lock()
		m.rooms[slug] = room
		m.chatHistory[slug] = make([]ws.ChatPayload, 0, chatHistorySize)
		m.mu.Unlock()

		log.Printf("room loaded from DB: slug=%s", slug)
		return room
	}

	return nil
}

func (m *Manager) handleJoin(hub *ws.Hub, cm ws.ClientMessage) {
	client := cm.Client
	slug := client.RoomSlug

	m.mu.RLock()
	room := m.rooms[slug]
	m.mu.RUnlock()
	if room == nil {
		return
	}

	// First client becomes host
	if room.GetHostID() == "" {
		room.SetHost(client.ID)
		client.IsHost = true
	}

	// Send room state to joining client
	state := room.GetState()
	peers := m.getPeers(slug)
	statePayload := ws.RoomStatePayload{
		VideoURL: state.URL,
		Playing:  state.Playing,
		Time:     state.Time,
		Peers:    peers,
		HostID:   room.GetHostID(),
		YourID:   client.ID,
	}
	payloadBytes, _ := json.Marshal(statePayload)
	hub.SendToClient(slug, client.ID, ws.Message{
		Type:    ws.TypeRoomState,
		Payload: payloadBytes,
	})

	// Send chat history
	m.mu.RLock()
	history := m.chatHistory[slug]
	m.mu.RUnlock()
	if len(history) > 0 {
		historyBytes, _ := json.Marshal(history)
		hub.SendToClient(slug, client.ID, ws.Message{
			Type:    ws.TypeChatHistory,
			Payload: historyBytes,
		})
	}

	// Broadcast updated peers to everyone
	m.broadcastPeers(slug)
}

func (m *Manager) handleLeave(hub *ws.Hub, cm ws.ClientMessage) {
	client := cm.Client
	slug := client.RoomSlug

	m.mu.RLock()
	room := m.rooms[slug]
	m.mu.RUnlock()
	if room == nil {
		return
	}

	// If host left, assign new host
	if room.GetHostID() == client.ID {
		clients := hub.GetClientsInRoom(slug)
		if len(clients) > 0 {
			newHost := clients[0]
			room.SetHost(newHost.ID)
			newHost.IsHost = true
			log.Printf("new host assigned: room=%s host=%s", slug, newHost.ID)
		}
	}

	m.broadcastPeers(slug)

	// Cleanup empty rooms from memory (keep in DB)
	if hub.RoomSize(slug) == 0 {
		m.mu.Lock()
		delete(m.rooms, slug)
		delete(m.chatHistory, slug)
		m.mu.Unlock()
		log.Printf("room unloaded from memory: slug=%s", slug)
	}
}

func (m *Manager) handlePlayerPlay(hub *ws.Hub, cm ws.ClientMessage) {
	var p ws.PlayerPayload
	json.Unmarshal(cm.Message.Payload, &p)

	room := m.GetRoom(cm.Client.RoomSlug)
	if room == nil {
		return
	}
	room.SetPlaying(true, p.Time)
	hub.BroadcastToRoom(cm.Client.RoomSlug, cm.Message, cm.Client.ID)
}

func (m *Manager) handlePlayerPause(hub *ws.Hub, cm ws.ClientMessage) {
	var p ws.PlayerPayload
	json.Unmarshal(cm.Message.Payload, &p)

	room := m.GetRoom(cm.Client.RoomSlug)
	if room == nil {
		return
	}
	room.SetPlaying(false, p.Time)
	hub.BroadcastToRoom(cm.Client.RoomSlug, cm.Message, cm.Client.ID)
}

func (m *Manager) handlePlayerSeek(hub *ws.Hub, cm ws.ClientMessage) {
	var p ws.PlayerPayload
	json.Unmarshal(cm.Message.Payload, &p)

	room := m.GetRoom(cm.Client.RoomSlug)
	if room == nil {
		return
	}
	room.SetTime(p.Time)
	hub.BroadcastToRoom(cm.Client.RoomSlug, cm.Message, cm.Client.ID)
}

func (m *Manager) handlePlayerSource(hub *ws.Hub, cm ws.ClientMessage) {
	var p ws.PlayerPayload
	json.Unmarshal(cm.Message.Payload, &p)

	room := m.GetRoom(cm.Client.RoomSlug)
	if room == nil {
		return
	}
	room.SetVideoURL(p.URL)

	// Persist video URL to DB
	if m.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := m.db.UpdateRoomVideoURL(ctx, cm.Client.RoomSlug, p.URL); err != nil {
			log.Printf("failed to update video URL in DB: %v", err)
		}
	}

	hub.BroadcastToRoom(cm.Client.RoomSlug, cm.Message, cm.Client.ID)
}

func (m *Manager) handlePlayerSync(hub *ws.Hub, cm ws.ClientMessage) {
	var p ws.PlayerPayload
	json.Unmarshal(cm.Message.Payload, &p)

	room := m.GetRoom(cm.Client.RoomSlug)
	if room == nil {
		return
	}
	room.SetTime(p.Time)
	hub.BroadcastToRoom(cm.Client.RoomSlug, cm.Message, cm.Client.ID)
}

func (m *Manager) handleChat(hub *ws.Hub, cm ws.ClientMessage) {
	var chat ws.ChatPayload
	json.Unmarshal(cm.Message.Payload, &chat)
	chat.Username = cm.Client.Username

	// Store in history
	m.mu.Lock()
	history := m.chatHistory[cm.Client.RoomSlug]
	if len(history) >= chatHistorySize {
		history = history[1:]
	}
	m.chatHistory[cm.Client.RoomSlug] = append(history, chat)
	m.mu.Unlock()

	// Re-marshal with server-set username
	payloadBytes, _ := json.Marshal(chat)
	msg := ws.Message{
		Type:    ws.TypeChatMessage,
		Payload: payloadBytes,
		From:    cm.Client.ID,
	}
	hub.BroadcastToRoom(cm.Client.RoomSlug, msg, "")
}

func (m *Manager) handleRTCRelay(hub *ws.Hub, cm ws.ClientMessage) {
	if cm.Message.To == "" {
		return
	}
	msg := cm.Message
	msg.From = cm.Client.ID
	hub.SendToClient(cm.Client.RoomSlug, cm.Message.To, msg)
}

func (m *Manager) handleScreenShare(hub *ws.Hub, cm ws.ClientMessage) {
	hub.BroadcastToRoom(cm.Client.RoomSlug, cm.Message, cm.Client.ID)
}

func (m *Manager) getPeers(slug string) []ws.PeerInfo {
	clients := m.hub.GetClientsInRoom(slug)
	peers := make([]ws.PeerInfo, 0, len(clients))
	for _, c := range clients {
		peers = append(peers, ws.PeerInfo{
			ID:       c.ID,
			Username: c.Username,
			IsHost:   c.IsHost,
		})
	}
	return peers
}

func (m *Manager) broadcastPeers(slug string) {
	peers := m.getPeers(slug)
	payload, _ := json.Marshal(ws.PeersPayload{Peers: peers})
	m.hub.BroadcastToRoom(slug, ws.Message{
		Type:    ws.TypeRoomPeers,
		Payload: payload,
	}, "")
}

func generateSlug() string {
	b := make([]byte, slugLength)
	for i := range b {
		b[i] = slugChars[rand.Intn(len(slugChars))]
	}
	return string(b)
}
