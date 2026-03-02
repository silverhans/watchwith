package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

const (
	writeTimeout = 10 * time.Second
	pongWait     = 60 * time.Second
	pingInterval = 30 * time.Second
	maxMsgSize   = 65536
)

type Client struct {
	ID       string
	Username string
	RoomSlug string
	IsHost   bool
	Hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	done     chan struct{}
	once     sync.Once
}

func NewClient(id, username, roomSlug string, isHost bool, hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		ID:       id,
		Username: username,
		RoomSlug: roomSlug,
		IsHost:   isHost,
		Hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		done:     make(chan struct{}),
	}
}

func (c *Client) ReadPump(ctx context.Context) {
	defer func() {
		c.Hub.Unregister <- c
		c.Close()
	}()

	c.conn.SetReadLimit(maxMsgSize)

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				log.Printf("read error client=%s: %v", c.ID, err)
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("invalid message from client=%s: %v", c.ID, err)
			continue
		}
		msg.From = c.ID

		c.Hub.Incoming <- ClientMessage{Client: c, Message: msg}
	}
}

func (c *Client) WritePump(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Write(writeCtx, websocket.MessageText, message)
			cancel()
			if err != nil {
				log.Printf("write error client=%s: %v", c.ID, err)
				return
			}
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Ping(pingCtx)
			cancel()
			if err != nil {
				log.Printf("ping error client=%s: %v", c.ID, err)
				return
			}
		case <-c.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) Send(data []byte) {
	select {
	case c.send <- data:
	default:
		log.Printf("send buffer full, dropping client=%s", c.ID)
		c.Close()
	}
}

func (c *Client) Close() {
	c.once.Do(func() {
		close(c.done)
		c.conn.Close(websocket.StatusNormalClosure, "")
	})
}

type ClientMessage struct {
	Client  *Client
	Message Message
}
