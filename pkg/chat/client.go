package chat

import (
	"bytes"
	"github.com/fasthttp/websocket"
	"log"
	"time"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client 1.
//   - Represents a single WebSocket client connected to the hub. It holds:

//- **`clients` map**: The `Hub` maintains a mapping of all active clients. This allows
//it to send broadcast messages to all clients' `Send` channels

type Client struct {
	Hub  *Hub
	Conn *websocket.Conn
	Send chan []byte //channel to send data each clients personal mailbox
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.Hub.broadcast <- message
	}

}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod) // for the send ing message and to keep the ws client alie
	defer func() {                       // this will run whenever the writePump func is return or closed
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				return
			}
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.Send)
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}

}

func PeerChatConn(c *websocket.Conn, hub *Hub) {
	client := &Client{
		Hub:  hub,
		Conn: c,
		Send: make(chan []byte, 256),
	}
	client.Hub.register <- client
	go client.WritePump() // core routine of this function
	client.ReadPump()
}

//#### Example Workflow:
//- A `Client` sends a message using `ReadPump`. It writes the message to `Hub.broadcast`.
//- The `Hub` listens to `broadcast` in its event loop (`Hub.Run`) and forwards the message to the
//`Send` channel of each client in its `clients` map.
//- Each client's `WritePump` receives the message from its `Send` channel and writes it to its WebSocket connection.
