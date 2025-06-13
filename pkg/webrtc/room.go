// Specifically, we manage a "room" where users can communicate with each other.

package webrtc

import (
	"encoding/json"
	"github.com/gofiber/websocket/v2"
	"github.com/kabenari/webrtc/pkg/chat"
	"github.com/pion/webrtc/v3"
	"log"
	"sync"
)

type Room struct {
	Peers *Peers
	Hub   *chat.Hub
}

// RoomConn handles the connection of a single user (peer) to the WebRTC room.
// It manages the creation of their communication channel, tracks (audio/video), and data exchange.

func RoomConn(c *websocket.Conn, p *Peers) {
	var config webrtc.Configuration
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return
	}
	defer peerConnection.Close()

	//Tell the peer connection that we want to receive both audio and video from the other users.

	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := peerConnection.AddTransceiverFromKind(
			typ, webrtc.RTPTransceiverInit{
				Direction: webrtc.RTPTransceiverDirectionRecvonly,
			}); err != nil {
			log.Println(err)
			return
		}
	}

	//Create a new peer state and keep track of it
	newPeer := PeerConnectionState{
		PeerConnection: peerConnection,
		websocket: &ThreadSafeWriter{
			Conn:  c,
			Mutex: sync.Mutex{},
		},
	}

	p.ListLock.Lock()
	p.Connections = append(p.Connections, newPeer)
	p.ListLock.Unlock()

	//Handle new ICE candidate events (used for connecting peers directly)
	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		candidateString, err := json.Marshal(i.ToJSON())
		if err != nil {
			return
		}

		// Send the ICE candidate to the peer through WebSocket
		if writeErr := newPeer.websocket.WriteJSON(&websocketMessage{
			Event: "candidate",
			Data:  string(candidateString),
		}); writeErr != nil {
			log.Println(writeErr)
		}
	})

	peerConnection.OnConnectionStateChange(func(pp webrtc.PeerConnectionState) {
		switch pp {
		case webrtc.PeerConnectionStateFailed:
			if err := peerConnection.Close(); err != nil {
				return
			}
		case webrtc.PeerConnectionStateClosed:
			p.SignalPeerConnections()
		}
	})

	//1.
	//- This handler is called every time a remote peer sends a media track (e.g., audio or video) to the peer connection.
	//- It processes the received RTP (Real-time Transport Protocol) packets from the remote track
	//Handle incoming audio/video tracks from a remote peer

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {

		// Try to add the new track to the room's peer handling
		trackLocal := p.AddTrack(track)
		if trackLocal == nil {
			return
		}
		defer p.RemoveTrack(trackLocal)

		// Create a buffer for RTP packets
		buf := make([]byte, 1500)

		//- If `p.AddTrack(track)` returns `nil`, it means adding the track failed (could be due to various
		//  reasons, like unsupported codec), and then no further processing occurs.
		//- If successfully added, it starts a loop to read incoming RTP packets from
		//  the remote track (`track.Read(buf)`) and forwards them to the "local track" using `trackLocal.Write(buf[:i])`.

		for {
			i, _, err := track.Read(buf)
			if err != nil {
				return
			}

			if _, err = trackLocal.Write(buf[:i]); err != nil {
				return
			}
		}
	})

	p.SignalPeerConnections()
	message := &websocketMessage{}
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			return
		} else if err := json.Unmarshal(raw, &message); err != nil {
			log.Println(err)
			return
		}

		switch message.Event {
		case "candidate":
			candidate := webrtc.ICECandidateInit{}
			if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
				log.Println(err)
				return
			}
			// Add the candidate to the WebRTC connection

			if err := peerConnection.AddICECandidate(candidate); err != nil {
				log.Println(err)
				return
			}

			// When we get an "answer" (acceptance for a WebRTC offer), set it
		case "answer":
			answer := webrtc.SessionDescription{}
			if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
				log.Println(err)
				return
			}
			if err := peerConnection.SetRemoteDescription(answer); err != nil {
				log.Println(err)
				return
			}
		}
	}

}
