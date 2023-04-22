package rtc

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/dmisol/sfu2/internal/bot"
	"github.com/dmisol/sfu2/internal/defs"
	"github.com/dmisol/sfu2/internal/media"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

func NewUser(room defs.Room, conf *defs.Conf, w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	runBot := r.URL.Query().Has("bot")

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	unsafeConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	c := &defs.ThreadSafeWriter{Conn: unsafeConn}

	defer c.Close()
	go func() {
		<-ctx.Done()
		log.Println("context timeout")
		c.Close()
	}()

	// Create new PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{}) // webrtc. //room.api.
	if err != nil {
		log.Print(err)
		return
	}

	defer peerConnection.Close() //nolint

	// Accept one audio and one video track incoming
	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := peerConnection.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Print(err)
			return
		}
	}

	// Add our new PeerConnection to global list
	room.AddPeerConnecrion(defs.PeerConnectionState{PeerConnection: peerConnection, Websocket: c})

	// Trickle ICE. Emit server candidate to client
	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		candidateString, err := json.Marshal(i.ToJSON())
		if err != nil {
			log.Println(err)
			return
		}

		if writeErr := c.WriteJSON(&websocketMessage{
			Event: "candidate",
			Data:  string(candidateString),
		}); writeErr != nil {
			log.Println(writeErr)
		}
	})

	// If PeerConnection is closed remove it from global list
	peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		switch p {
		case webrtc.PeerConnectionStateFailed:
			if err := peerConnection.Close(); err != nil {
				log.Print(err)
			}
		case webrtc.PeerConnectionStateClosed:
			room.SignalPeerConnections()
		}
	})

	var mediaProc defs.Media
	if runBot {
		aiBot := bot.NewBot(ctx, conf.BotUrl) // to enambe bot act as a peer
		mediaProc = media.NewRegularMedia(room, aiBot)
	} else {
		mediaProc = media.NewRegularMedia(room, nil)
	}

	peerConnection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		log.Println(t.Codec(), t.ID(), t.StreamID())
		if t.Kind() == webrtc.RTPCodecTypeAudio {
			mediaProc.OnAudioTrack(ctx, t)
		} else {
			mediaProc.OnVideoTrack(ctx, t)
		} // Create a track to fan out our incoming video to all peers
	})

	// Signal for the new PeerConnection
	room.SignalPeerConnections()

	message := &websocketMessage{}
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			log.Println(err)
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

			if err := peerConnection.AddICECandidate(candidate); err != nil {
				log.Println(err)
				return
			}
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
