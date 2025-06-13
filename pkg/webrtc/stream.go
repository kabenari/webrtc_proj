package webrtc

import (
	"encoding/json"
	"github.com/gofiber/websocket/v2"
	"github.com/pion/webrtc/v3"
	"sync"
)

//It primarily handles signaling, peer connection setup,
//and the transmission of ICE candidates and SDP offers/answers.

func StreamConn(c *websocket.Conn, p *Peers) {
	var config webrtc.Configuration
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return
	}
	defer peerConnection.Close()

	//so basically adding two transceivers one for audio and one for video Direction: webrtc.RTPTransceiverDirectionSendonly,
	//this will set the direction of one way flow only one direction

	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := peerConnection.AddTransceiverFromKind(
			typ, webrtc.RTPTransceiverInit{
				Direction: webrtc.RTPTransceiverDirectionSendonly,
			}); err != nil {
			return
		}
	}

	//now making a new peer

	newPeer := PeerConnectionState{
		PeerConnection: peerConnection,
		websocket: &ThreadSafeWriter{
			Conn:  c,
			Mutex: sync.Mutex{},
		},
	}

	//update the connections map in the peer

	p.ListLock.Lock()
	p.Connections = append(p.Connections, newPeer)
	p.ListLock.Unlock()

	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		candidateString, err := json.Marshal(i.ToJSON())
		if err != nil {
			return
		}

		if writeErr := newPeer.websocket.WriteJSON(&websocketMessage{
			Event: "candidate",
			Data:  string(candidateString),
		}); writeErr != nil {
			return
		}
	})

	//this function is used to verify the connectio if peer changes state return from the function

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

	p.SignalPeerConnections()

	//This code is part of a WebRTC signaling implementation using WebSockets
	//It handles signaling messages received through a WebSocket connection to
	//exchange information required for establishing a WebRTC peer-to-peer connection

	message := &websocketMessage{}
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			return
		} else if err := json.Unmarshal(raw, message); err != nil {
			return
		}

		switch message.Event {
		//This part of the code processes ICE candidates sent by
		//the remote peer and adds them to the local WebRTC connection.
		case "candidate":
			candidate := webrtc.ICECandidateInit{}

			//this places are err = becasue this function just returns a error thats all
			if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
				return
			}
			if err := peerConnection.AddICECandidate(candidate); err != nil {
				return
			}

		//The answer finalizes the WebRTC connection setup by specifying
		//the remote peer’s media configuration.
		case "answer":
			answer := webrtc.SessionDescription{}
			if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
				return
			}
			//Sets the remote peer's media configuration for the local WebRTC connection.
			if err := peerConnection.SetRemoteDescription(answer); err != nil {
				return
			}
		}
	}

}
