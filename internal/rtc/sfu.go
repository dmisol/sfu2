package rtc

import (
	"log"
	"net"

	"github.com/pion/webrtc/v3"
)

func NewApi() (*webrtc.API, error) {
	m := webrtc.MediaEngine{}

	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/H264", ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        126,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		log.Println("reg videoo", err)
		return nil, err
	}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/opus", ClockRate: 48000, Channels: 2, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		log.Println("reg audio", err)
		return nil, err
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
		return nil, err
	}

	tcpMux := webrtc.NewICETCPMux(nil, tcpListener, 8)
	settingEngine.SetICETCPMux(tcpMux)

	udpListener, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IP{0, 0, 0, 0},
		Port: 3478,
	})
	if err != nil {
		log.Println("listenUDP()", err)
		return nil, err
	}

	udpMux := webrtc.NewICEUDPMux(nil, udpListener)
	settingEngine.SetICEUDPMux(udpMux)

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(&m),
		webrtc.WithSettingEngine(settingEngine),
	)
	return api, nil
}
