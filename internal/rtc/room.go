package rtc

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/dmisol/sfu2/internal/defs"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

const timeout = 2 * time.Hour

type websocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

func NewRoom() *Room {
	r := &Room{
		trackLocals: make(map[string]*webrtc.TrackLocalStaticRTP),
	}

	m := webrtc.MediaEngine{}

	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/H264", ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        126,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		log.Println("reg videoo", err)
		return nil
	}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/opus", ClockRate: 48000, Channels: 2, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		log.Println("reg audio", err)
		return nil
	}
	settingEngine := webrtc.SettingEngine{}

	// Enable support only for TCP ICE candidates.
	settingEngine.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeTCP4,
		//		webrtc.NetworkTypeUDP4,
		//webrtc.NetworkTypeTCP6,
	})

	tcpListener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IP{0, 0, 0, 0},
		Port: 3478,
	})

	if err != nil {
		log.Println("listenTCP()", err)
		return nil
	}

	tcpMux := webrtc.NewICETCPMux(nil, tcpListener, 8)
	settingEngine.SetICETCPMux(tcpMux)

	/*
		udpListener, err := net.ListenUDP("udp", &net.UDPAddr{
			IP:   net.IP{0, 0, 0, 0},
			Port: MediaPort,
		})
		if err != nil {
			log.Println("listenUDP()", err)
			return
		}

		udpMux := webrtc.NewICEUDPMux(nil, udpListener)
		settingEngine.SetICEUDPMux(udpMux)
	*/

	r.api = webrtc.NewAPI(
		webrtc.WithMediaEngine(&m),
		webrtc.WithSettingEngine(settingEngine),
	)

	// request a keyframe every 3 seconds
	go func() {
		for range time.NewTicker(time.Second * 3).C {
			r.dispatchKeyFrame()
		}
	}()

	return r
}

type Room struct {
	api *webrtc.API

	// lock for peerConnections and trackLocals
	mu              sync.RWMutex
	peerConnections []defs.PeerConnectionState
	trackLocals     map[string]*webrtc.TrackLocalStaticRTP
}

// Add to list of tracks and fire renegotation for all PeerConnections
func (room *Room) AddTrack(t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	room.mu.Lock()
	defer func() {
		room.mu.Unlock()
		room.SignalPeerConnections()
	}()

	// Create a new TrackLocal with the same codec as our incoming
	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}

	room.trackLocals[t.ID()] = trackLocal
	return trackLocal
}

// Remove from list of tracks and fire renegotation for all PeerConnections
func (room *Room) RemoveTrack(t *webrtc.TrackLocalStaticRTP) {
	room.mu.Lock()
	defer func() {
		room.mu.Unlock()
		room.SignalPeerConnections()
	}()

	delete(room.trackLocals, t.ID())
}

// SignalPeerConnections updates each PeerConnection so that it is getting all the expected media tracks
func (room *Room) SignalPeerConnections() {
	room.mu.Lock()
	defer func() {
		room.mu.Unlock()
		room.dispatchKeyFrame()
	}()

	attemptSync := func() (tryAgain bool) {
		for i := range room.peerConnections {
			if room.peerConnections[i].PeerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				room.peerConnections = append(room.peerConnections[:i], room.peerConnections[i+1:]...)
				return true // We modified the slice, start from the beginning
			}

			// map of sender we already are seanding, so we don't double send
			existingSenders := map[string]bool{}

			for _, sender := range room.peerConnections[i].PeerConnection.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				// If we have a RTPSender that doesn't map to a existing track remove and signal
				if _, ok := room.trackLocals[sender.Track().ID()]; !ok {
					if err := room.peerConnections[i].PeerConnection.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}

			// Don't receive videos we are sending, make sure we don't have loopback
			for _, receiver := range room.peerConnections[i].PeerConnection.GetReceivers() {
				if receiver.Track() == nil {
					continue
				}

				existingSenders[receiver.Track().ID()] = true
			}

			// Add all track we aren't sending yet to the PeerConnection
			for trackID := range room.trackLocals {
				if _, ok := existingSenders[trackID]; !ok {
					if _, err := room.peerConnections[i].PeerConnection.AddTrack(room.trackLocals[trackID]); err != nil {
						return true
					}
				}
			}

			offer, err := room.peerConnections[i].PeerConnection.CreateOffer(nil)
			if err != nil {
				return true
			}

			if err = room.peerConnections[i].PeerConnection.SetLocalDescription(offer); err != nil {
				return true
			}

			offerString, err := json.Marshal(offer)
			if err != nil {
				return true
			}

			if err = room.peerConnections[i].Websocket.WriteJSON(&websocketMessage{
				Event: "offer",
				Data:  string(offerString),
			}); err != nil {
				return true
			}
		}

		return
	}

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			// Release the lock and attempt a sync in 3 seconds. We might be blocking a RemoveTrack or AddTrack
			go func() {
				time.Sleep(time.Second * 3)
				room.SignalPeerConnections()
			}()
			return
		}

		if !attemptSync() {
			break
		}
	}
}

// dispatchKeyFrame sends a keyframe to all PeerConnections, used everytime a new user joins the call
func (room *Room) dispatchKeyFrame() {
	room.mu.Lock()
	defer room.mu.Unlock()

	for i := range room.peerConnections {
		for _, receiver := range room.peerConnections[i].PeerConnection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = room.peerConnections[i].PeerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}
}

func (room *Room) AddPeerConnecrion(pcs defs.PeerConnectionState) {
	room.mu.Lock()
	defer room.mu.Unlock()

	room.peerConnections = append(room.peerConnections, pcs)
}

// Handle incoming websockets
func (room *Room) WebsocketHandler(w http.ResponseWriter, r *http.Request) {

	runBot := r.URL.Query().Has("bot")
	ftar := r.URL.Query().Get("ftar")
	log.Println(ftar, runBot)

	NewUser(room, w, r)
}
