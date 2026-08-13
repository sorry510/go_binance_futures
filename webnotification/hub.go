package webnotification

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second
	maxMessage = 1024
)

type client struct {
	connection *websocket.Conn
	send       chan []byte
}

type hub struct {
	clients    map[*client]struct{}
	register   chan *client
	unregister chan *client
	broadcast  chan []byte
}

var (
	defaultHub     *hub
	defaultHubOnce sync.Once
)

func getHub() *hub {
	defaultHubOnce.Do(func() {
		defaultHub = &hub{
			clients:    make(map[*client]struct{}),
			register:   make(chan *client),
			unregister: make(chan *client),
			broadcast:  make(chan []byte, 256),
		}
		go defaultHub.run()
	})
	return defaultHub
}

func (h *hub) run() {
	for {
		select {
		case current := <-h.register:
			h.clients[current] = struct{}{}
		case current := <-h.unregister:
			if _, ok := h.clients[current]; ok {
				delete(h.clients, current)
				close(current.send)
			}
		case message := <-h.broadcast:
			for current := range h.clients {
				select {
				case current.send <- message:
				default:
					delete(h.clients, current)
					close(current.send)
				}
			}
		}
	}
}

func ServeConnection(connection *websocket.Conn) {
	current := &client{connection: connection, send: make(chan []byte, 32)}
	h := getHub()
	h.register <- current
	go current.writePump()
	current.readPump(h)
}

func Broadcast(eventType string, data any) error {
	payload, err := json.Marshal(map[string]any{
		"type": eventType,
		"data": data,
	})
	if err != nil {
		return err
	}

	h := getHub()
	select {
	case h.broadcast <- payload:
	default:
	}
	return nil
}

func (current *client) readPump(h *hub) {
	defer func() {
		h.unregister <- current
		_ = current.connection.Close()
	}()
	current.connection.SetReadLimit(maxMessage)
	_ = current.connection.SetReadDeadline(time.Now().Add(pongWait))
	current.connection.SetPongHandler(func(string) error {
		return current.connection.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := current.connection.ReadMessage(); err != nil {
			return
		}
	}
}

func (current *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = current.connection.Close()
	}()
	for {
		select {
		case message, ok := <-current.send:
			_ = current.connection.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = current.connection.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := current.connection.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = current.connection.SetWriteDeadline(time.Now().Add(writeWait))
			if err := current.connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
