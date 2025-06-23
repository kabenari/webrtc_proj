package handlers

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	w "github.com/kabenari/webrtc/pkg/webrtc"
	"os"
	"time"
)

func Stream(c *fiber.Ctx) error {
	suuid := c.Params("suuid")
	if suuid == "" {
		return nil
	}
	ws := "ws"
	if os.Getenv("ENV") == "PROD" {
		ws = "wss"
	}
	w.RoomsLock.Lock()
	_, ok := w.Rooms[suuid]
	w.RoomsLock.Unlock()
	if ok {
		return c.JSON(fiber.Map{
			"streamWebsocketAddr": fmt.Sprintf("%s://%s/stream/%s/websocket", ws, c.Hostname(), suuid),
			"chatWebsocketAddr":   fmt.Sprintf("%s://%s/stream/%s/chat/websocket", ws, c.Hostname(), suuid),
			"viewerWebSocketAddr": fmt.Sprintf("%s://%s/stream/%s/viewer/websocket", ws, c.Hostname(), suuid),
		})
	}
	return c.Status(404).JSON(fiber.Map{
		"error": "Stream not found",
	})
}

func StreamWebsocket(c *websocket.Conn) {
	suuid := c.Params("suuid")
	if suuid == "" {
		return
	}
	w.RoomsLock.Lock()
	if stream, ok := w.Streams[suuid]; ok {
		w.RoomsLock.Unlock()
		w.StreamConn(c, stream.Peers)
		return
	}
	w.RoomsLock.Unlock()
}

func StreamViewerWebsocket(c *websocket.Conn) {
	suuid := c.Params("suuid")
	if suuid == "" {
		return
	}
	w.RoomsLock.Lock()
	if stream, ok := w.Streams[suuid]; ok {
		w.RoomsLock.Unlock()
		viewerConn(c, stream.Peers)
		return
	}
	w.RoomsLock.Unlock()
}

func viewerConn(c *websocket.Conn, p *w.Peers) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	defer c.Close()

	for {
		select {
		case <-ticker.C:
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write([]byte(fmt.Sprintf("%d", len(p.Connections))))
		}
	}
}
