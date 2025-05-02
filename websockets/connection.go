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
	writeWait      = 5 * time.Second
	pongWait       = 100 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 8192
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
		send:    make(chan []byte, 256),
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
	// defer func() {
	// 	if r := recover(); r != nil {
	// 			c.Close()
	// 			err = ErrorConnectionClosed
	// 	}
	// }()

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
			c.Close()
			return ErrorBufferFull
		}
	}
}

func (c *Connection) readPump() {
	defer c.Close()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				log.Printf("error during websocket pump: %v", err)
			}
			c.Close()
			break
		}

		if c.handler != nil {
			c.handler(message)
		}
	}
}

func (c *Connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-c.closed:
			return
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(ws.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(ws.BinaryMessage)
			if err != nil {
				return
			}

			_, err = w.Write(message)

			if err != nil {
				w.Close()
				return
			}

			for range len(c.send) {
				_, err = w.Write(<-c.send)
				if err != nil {
					w.Close()
					return
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(ws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

var (
	ErrorConnectionClosed = fmt.Errorf("Connection closed")
	ErrorBufferFull       = fmt.Errorf("Send buffer full")
)
