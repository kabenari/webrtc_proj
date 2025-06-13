package handlers

import (
	"crypto/sha256"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/kabenari/webrtc/pkg/chat"
	w "github.com/kabenari/webrtc/pkg/webrtc"
	"github.com/pion/webrtc/v3"
	"os"
	"time"
)

type websocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

func RoomCreate(c *fiber.Ctx) error {
	return c.Redirect(fmt.Sprintf("/room/%s", uuid.New().String()))
}

func Room(c *fiber.Ctx) error {
	uuid := c.Params("uuid")
	if uuid == "" {
		c.Status(404)
		return nil
	}
	ws := "ws"
	if os.Getenv("ENV") == "PROD" {
		ws = "wss"
	}
	uuid, suuid, _ := createOrGetRoom(uuid)
	return c.Render(
		"peer", fiber.Map{
			"RoomWebSocketAddr":   fmt.Sprintf("%s://%s/room/%s/websocket", ws, c.Hostname(), uuid),
			"RoomLink":            fmt.Sprintf("%s://%s/room/%s", c.Protocol(), c.Hostname(), uuid),
			"ChatWebSocketAddr":   fmt.Sprintf("%s://%s/room/%s/chat/websocket", ws, c.Hostname(), uuid),
			"ViewerWebSocketAddr": fmt.Sprintf("%s://%s/room/%s/viewer/websocket", ws, c.Hostname(), uuid),
			"StreamLink":          fmt.Sprintf("%s://%s/stream/%s", c.Protocol(), c.Hostname(), suuid),
			"Type":                "room",
		}, "layouts/main")
}

func RoomWebsocket(c *websocket.Conn) {
	uuid := c.Params("uuid")
	if uuid == "" {
		return
	}

	_, _, room := createOrGetRoom(uuid)
	w.RoomConn(c, room.Peers)
}

//the code is a crucial helper method designed to either **create a new room** or **retrieve an
//existing one**. It ensures that a room is available and properly initialized before any operations
//(such as WebSocket connections) can be carried out for that room. Below is a thorough breakdown of
//this method, including all processes and concepts involved.

func createOrGetRoom(uuid string) (string, string, *w.Room) {
	w.RoomsLock.Lock()
	defer w.RoomsLock.Unlock()
	h := sha256.New()
	h.Write([]byte(uuid))
	suuid := fmt.Sprintf("%x", h.Sum(nil))

	if room := w.Rooms[uuid]; room != nil {
		if _, ok := w.Streams[suuid]; !ok {
			w.Streams[suuid] = room
		}
	}
	hub := chat.NewHub()
	p := &w.Peers{}
	p.TrackLocals = make(map[string]*webrtc.TrackLocalStaticRTP)
	room := &w.Room{
		Peers: p,
		Hub:   hub,
	}
	w.Rooms[uuid] = room
	w.Streams[suuid] = room
	go hub.Run()
	return uuid, suuid, room
}

// - Handles WebSocket connections for viewers in a room.
func RoomViewerWebsocket(c *websocket.Conn) {
	uuid := c.Params("uuid")
	if uuid == "" {
		return
	}
	w.RoomsLock.Lock()
	if peer, ok := w.Rooms[uuid]; ok {
		w.RoomsLock.Unlock()
		RoomViewerConn(c, peer.Peers)
		return
	}
	w.RoomsLock.Unlock()
}

//- **Purpose**: Periodically sends the number of active connections to the viewer via WebSocket.
//1. Creates a 1-second ticker.
//2. On each tick:
//	- Writes the current number of active peers (`len(p.Connections)`) to the WebSocket.
//	- Stops when the connection closes.

func RoomViewerConn(c *websocket.Conn, p *w.Peers) {
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
