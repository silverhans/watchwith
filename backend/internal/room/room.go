package room

import (
	"sync"
	"time"
)

type VideoState struct {
	URL     string  `json:"url"`
	Playing bool    `json:"playing"`
	Time    float64 `json:"time"`
	Updated int64   `json:"updated"`
}

type Room struct {
	ID        string     `json:"id"`
	Slug      string     `json:"slug"`
	Name      string     `json:"name"`
	HostID    string     `json:"hostId"`
	Video     VideoState `json:"video"`
	CreatedAt time.Time  `json:"createdAt"`
	mu        sync.RWMutex
}

func NewRoom(id, slug, name, hostID string) *Room {
	return &Room{
		ID:        id,
		Slug:      slug,
		Name:      name,
		HostID:    hostID,
		Video:     VideoState{},
		CreatedAt: time.Now(),
	}
}

func (r *Room) SetVideoURL(url string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Video.URL = url
	r.Video.Time = 0
	r.Video.Playing = false
	r.Video.Updated = time.Now().UnixMilli()
}

func (r *Room) SetPlaying(playing bool, t float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Video.Playing = playing
	r.Video.Time = t
	r.Video.Updated = time.Now().UnixMilli()
}

func (r *Room) SetTime(t float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Video.Time = t
	r.Video.Updated = time.Now().UnixMilli()
}

func (r *Room) GetState() VideoState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Video
}

func (r *Room) SetHost(hostID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.HostID = hostID
}

func (r *Room) GetHostID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.HostID
}
