package ws

import (
	"encoding/json"
	"log"
	"sync"
)

type Hub struct {
	rooms      map[string]map[*Client]bool
	mu         sync.RWMutex
	Register   chan *Client
	Unregister chan *Client
	Incoming   chan ClientMessage
	handlers   map[MessageType]MessageHandler
}

type MessageHandler func(hub *Hub, cm ClientMessage)

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		Register:   make(chan *Client, 64),
		Unregister: make(chan *Client, 64),
		Incoming:   make(chan ClientMessage, 256),
		handlers:   make(map[MessageType]MessageHandler),
	}
}

func (h *Hub) OnMessage(msgType MessageType, handler MessageHandler) {
	h.handlers[msgType] = handler
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if _, ok := h.rooms[client.RoomSlug]; !ok {
				h.rooms[client.RoomSlug] = make(map[*Client]bool)
			}
			h.rooms[client.RoomSlug][client] = true
			h.mu.Unlock()
			log.Printf("client registered: id=%s room=%s username=%s", client.ID, client.RoomSlug, client.Username)

		case client := <-h.Unregister:
			h.mu.Lock()
			if clients, ok := h.rooms[client.RoomSlug]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.rooms, client.RoomSlug)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("client unregistered: id=%s room=%s", client.ID, client.RoomSlug)

			if handler, ok := h.handlers[TypeRoomLeave]; ok {
				handler(h, ClientMessage{Client: client, Message: Message{Type: TypeRoomLeave, From: client.ID}})
			}

		case cm := <-h.Incoming:
			if handler, ok := h.handlers[cm.Message.Type]; ok {
				handler(h, cm)
			} else {
				log.Printf("unhandled message type: %s from client=%s", cm.Message.Type, cm.Client.ID)
			}
		}
	}
}

func (h *Hub) BroadcastToRoom(slug string, msg Message, excludeID string) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("marshal error: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[slug]
	if !ok {
		return
	}
	for client := range clients {
		if client.ID != excludeID {
			client.Send(data)
		}
	}
}

func (h *Hub) SendToClient(slug, clientID string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("marshal error: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[slug]
	if !ok {
		return
	}
	for client := range clients {
		if client.ID == clientID {
			client.Send(data)
			return
		}
	}
}

func (h *Hub) GetClientsInRoom(slug string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[slug]
	if !ok {
		return nil
	}
	result := make([]*Client, 0, len(clients))
	for c := range clients {
		result = append(result, c)
	}
	return result
}

func (h *Hub) RoomSize(slug string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[slug])
}
