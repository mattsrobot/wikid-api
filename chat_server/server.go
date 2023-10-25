package chatserver

import (
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
)

type Client struct {
	IsClosing bool
	Lp        time.Time
	Mu        sync.Mutex
	Topics    map[string]bool
}

type Message struct {
	UserID     uint64
	Topic      string
	Connection *websocket.Conn
}

type Broadcast struct {
	Message string `json:"message"`
	Topic   string `json:"topic"`
}

type Echo struct {
	Message    string
	Connection *websocket.Conn
}

type Server struct {
	Clients     map[*websocket.Conn]*Client
	Subscribe   chan Message
	Unsubscribe chan Message
	Echo        chan Echo
	Broadcast   chan Broadcast
	Register    chan *websocket.Conn
	Unregister  chan *websocket.Conn
}
