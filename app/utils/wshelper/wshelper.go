package wshelper

import (
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type Event struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type NoticePayload struct {
	ID         uint       `json:"id"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Category   string     `json:"category"`
	SentAt     *time.Time `json:"sentAt"`
	IsRead     int8       `json:"isRead"`
	SenderName string     `json:"senderName"`
}

type Manager struct {
	mu      sync.RWMutex
	clients map[uint]map[*websocket.Conn]struct{}
}

var defaultManager = &Manager{
	clients: make(map[uint]map[*websocket.Conn]struct{}),
}

func DefaultManager() *Manager {
	return defaultManager
}

func (m *Manager) Register(userID uint, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.clients[userID]; !ok {
		m.clients[userID] = make(map[*websocket.Conn]struct{})
	}
	m.clients[userID][conn] = struct{}{}
}

func (m *Manager) Unregister(userID uint, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conns, ok := m.clients[userID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(m.clients, userID)
		}
	}
	_ = conn.Close()
}

func (m *Manager) SendToUser(userID uint, event Event) {
	m.mu.RLock()
	conns := make([]*websocket.Conn, 0)
	for conn := range m.clients[userID] {
		conns = append(conns, conn)
	}
	m.mu.RUnlock()

	for _, conn := range conns {
		if err := websocket.JSON.Send(conn, event); err != nil {
			m.Unregister(userID, conn)
		}
	}
}

func (m *Manager) BroadcastNotice(userIDs []uint, payload NoticePayload) {
	event := Event{Event: "notice:new", Data: payload}
	for _, userID := range userIDs {
		m.SendToUser(userID, event)
	}
}

func (m *Manager) SendConnected(userID uint) {
	m.SendToUser(userID, Event{
		Event: "notice:connected",
		Data: map[string]any{
			"connectedAt": time.Now(),
		},
	})
}
