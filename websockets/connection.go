package websockets

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
)

const (
	writeWait      = 0
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// MessageHandler defines a function that processes binary messaeges
type MessageHandler func([]byte)

// MessageSender defines an interface for sending messages
type MessageSender interface {
	SendBinary(data []byte) error
	Close()
}

type Connection struct {
	conn      *ws.Conn
	send      chan []byte
	handler   MessageHandler
	closeOnce sync.Once
	closed    chan struct{}
}

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// TODO: actually check the origin
	CheckOrigin: func(r *http.Request) bool { return true },
}

func Upgrade(w http.ResponseWriter, r *http.Request, handler MessageHandler) (*Connection, error) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		return nil, err
	}

	c := &Connection{
		conn:    conn,
		send:    make(chan []byte, 2048),
		handler: handler,
		closed:  make(chan struct{}),
	}

	go c.readPump()
	go c.writePump()

	return c, nil
}

func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		close(c.send)
		c.conn.Close()
	})
}

func (c *Connection) IsClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}

func (c *Connection) SendBinary(data []byte) (err error) {
	select {
	case <-c.closed:
		log.Printf("connection closed, returning error")
		return ErrorConnectionClosed
	default:
		select {
		case c.send <- data:
			return nil
		default:
			// Buffer probably full
			// c.Close()
			return ErrorBufferFull
		}
	}
}

func (c *Connection) readPump() {
	defer c.Close()
	var tzero time.Time
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(tzero)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(tzero)
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				log.Printf("error during websocket pump: %v", err)
			}
			c.Close()
			return
		}

		if c.handler != nil {
			c.handler(message)
		}
	}
}

func (c *Connection) writePump() {
	var tzero time.Time
	// ticker := time.NewTicker(pingPeriod)
	defer func() {
		// ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(tzero)
			if !ok {
				c.conn.WriteMessage(ws.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(ws.BinaryMessage)
			if err != nil {
				return
			}

			w.Write(message)

			for range len(c.send) {
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				log.Printf("error while closing writepump %v", err)
				return
			}

		case <-c.closed:
			return

		// case <-ticker.C:
		// 	log.Printf("ticker clock")
		// 	c.conn.SetWriteDeadline(tzero)
		// 	if err := c.conn.WriteMessage(ws.PingMessage, nil); err != nil {
		// 		return
		// 	}
		}
	}
}

var (
	ErrorConnectionClosed = fmt.Errorf("Connection closed")
	ErrorBufferFull       = fmt.Errorf("Send buffer full")
)
