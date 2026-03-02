package ws

import "encoding/json"

type MessageType string

const (
	// Room events
	TypeRoomJoin  MessageType = "room:join"
	TypeRoomLeave MessageType = "room:leave"
	TypeRoomState MessageType = "room:state"
	TypeRoomPeers MessageType = "room:peers"

	// Player events
	TypePlayerPlay   MessageType = "player:play"
	TypePlayerPause  MessageType = "player:pause"
	TypePlayerSeek   MessageType = "player:seek"
	TypePlayerSource MessageType = "player:source"
	TypePlayerSync   MessageType = "player:sync"

	// Chat events
	TypeChatMessage MessageType = "chat:message"
	TypeChatHistory MessageType = "chat:history"

	// WebRTC signaling
	TypeRTCOffer  MessageType = "rtc:offer"
	TypeRTCAnswer MessageType = "rtc:answer"
	TypeRTCICE    MessageType = "rtc:ice"

	// Screen sharing
	TypeScreenStart MessageType = "screen:start"
	TypeScreenStop  MessageType = "screen:stop"
)

type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
	From    string          `json:"from,omitempty"`
	To      string          `json:"to,omitempty"`
}

type JoinPayload struct {
	Username string `json:"username"`
}

type PlayerPayload struct {
	Time float64 `json:"time"`
	URL  string  `json:"url,omitempty"`
}

type ChatPayload struct {
	Text     string `json:"text"`
	Username string `json:"username"`
	Time     int64  `json:"time"`
}

type PeersPayload struct {
	Peers []PeerInfo `json:"peers"`
}

type PeerInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	IsHost   bool   `json:"isHost"`
}

type RoomStatePayload struct {
	VideoURL  string     `json:"videoUrl"`
	Playing   bool       `json:"playing"`
	Time      float64    `json:"time"`
	Peers     []PeerInfo `json:"peers"`
	HostID    string     `json:"hostId"`
	YourID    string     `json:"yourId"`
}
