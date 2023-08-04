package defs

import (
	"context"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

const DebugVideo = false

type Room interface {
	AddPeerConnecrion(PeerConnectionState)
	SignalPeerConnections(*int32)
	GetAPI() *webrtc.API

	AddTrack(*webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP         // inherited - creates track from remote
	AddSyntheticTrack(trackLocal webrtc.TrackLocal, pliNeeded *int32) // just appends track local
	RemoveTrack(webrtc.TrackLocal)
}

type Media interface {
	OnAudioTrack(context.Context, *webrtc.TrackRemote)
	OnVideoTrack(context.Context, *webrtc.TrackRemote)
}

type PeerConnectionState struct {
	PeerConnection *webrtc.PeerConnection
	Websocket      *ThreadSafeWriter
}

// Helper to make Gorilla Websockets threadsafe
type ThreadSafeWriter struct {
	*websocket.Conn
	mu sync.Mutex
}

func (t *ThreadSafeWriter) WriteJSON(v interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.Conn.WriteJSON(v)
}
