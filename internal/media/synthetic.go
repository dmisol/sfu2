package media

// #cgo linux CFLAGS: -I/usr/include/opus
// #cgo linux LDFLAGS: -L/usr/lib/x86_64-linux-gnu -lopus
// #include <opus.h>
import "C"

import (
	"io"
	"log"
	"time"

	"github.com/dmisol/sfu2/internal/media/opus"
	"github.com/pion/webrtc/v3"
	ms "github.com/pion/webrtc/v3/pkg/media"
)

const (
	bytes20ms = 1920
)

func (m *RegularMedia) RunPcmTrack(stmid string, tid string, audio io.Reader) {
	// 1. create trackLocalStaticRTP for bot's responses

	// TODO: force ptime ?
	t, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:     "audio/opus",
			ClockRate:    48000,
			Channels:     2,
			SDPFmtpLine:  "ptime=20", //"minptime=10;useinbandfec=1",
			RTCPFeedback: nil,
		}, tid, stmid)
	if err != nil {
		log.Println("local track creating", err)
		return
	}

	// 2. m.room.AddSyntheticTrack()
	m.room.AddSyntheticTrack(t)
	log.Println("ingress track added", t.ID())
	defer func() {
		m.room.RemoveTrack(t)
		log.Println("ingress track removed", t.ID())
	}()

	// 3. encoder
	enc := opus.NewOpusEncoder()
	defer enc.Close()

	var in []byte
	rd := make([]byte, 10000)
	last := time.Now()
	for {
		// 4. { read, encode, write to trackLocal }

		i, err := audio.Read(rd)
		if err != nil {
			log.Println("synth rd", err)
		}
		if i > 0 && err == nil {
			in = append(in, rd[:i]...)
			//log.Println("appended; in", i, len(in))
		}

		now := time.Now()
		var x []byte
		if (len(in) > bytes20ms) && now.Add(-19900*time.Microsecond).After(last) {
			last = now

			x, in = in[:bytes20ms], in[bytes20ms:]
			encoded, err := enc.Encode(x)
			if err != nil {
				log.Println("audio encoding error", err)
				return
			}

			if err := t.WriteSample(ms.Sample{Data: encoded, Duration: 20 * time.Millisecond}); err != nil {
				log.Println("synthetic write error", err)
			}
			//log.Println("sent; in", len(in))
		}
	}
}
