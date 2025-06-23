package handlers

import (
	"github.com/gofiber/websocket/v2"
	"github.com/kabenari/webrtc/pkg/chat"
	w "github.com/kabenari/webrtc/pkg/webrtc"
)

// RoomChatWebsocket establishes a WebSocket connection for chatting inside a room.
//
// Parameters:
//   - c: Fiber WebSocket connection object.
//
// Behavior:
//   - Extracts the "uuid" parameter from the WebSocket URL to identify the room.
//   - Looks up the room in the global `w.Rooms` map (with proper synchronization).
//   - If the room and its hub exist, it connects the WebSocket to the room's hub.

func RoomChatWebsocket(c *websocket.Conn) {
	uuid := c.Params("uuid")
	if uuid == "" {
		return
	}
	w.RoomsLock.Lock()
	room := w.Rooms[uuid]
	w.RoomsLock.Unlock()
	if room == nil {
		return
	}
	if room.Hub == nil {
		return
	}
	chat.PeerChatConn(c.Conn, room.Hub)
}

// StreamChatWebsocket establishes a WebSocket connection for chatting inside a stream.
//
// Parameters:
//   - c: Fiber WebSocket connection object.
//
// Behavior:
//   - Extracts the "suuid" parameter from the WebSocket URL to identify the stream.
//   - Looks up the stream in the global `w.Streams` map (with proper synchronization).
//   - If the stream doesn't have a hub, creates and starts one.
//   - Connects the WebSocket to the stream's hub for real-time communication.

func StreamChatWebsocket(c *websocket.Conn) {
	suuid := c.Params("suuid")
	if suuid == "" {
		return
	}
	w.RoomsLock.Lock()
	if stream, ok := w.Streams[suuid]; ok {
		if stream.Hub == nil {
			hub := chat.NewHub()
			stream.Hub = hub
			go hub.Run()
		}
		chat.PeerChatConn(c.Conn, stream.Hub)
		return
	}
	w.RoomsLock.Unlock()
}
