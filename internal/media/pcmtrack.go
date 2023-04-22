package media

// #cgo linux CFLAGS: -I/usr/include/opus
// #cgo linux LDFLAGS: -L/usr/lib/x86_64-linux-gnu -lopus
// #include <opus.h>
import "C"

import (
	"context"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dmisol/sfu2/internal/media/opus"
	"github.com/pion/webrtc/v3"
	ms "github.com/pion/webrtc/v3/pkg/media"
)

const (
	bytes20ms  = 1920
	filledFifo = 3 * bytes20ms
)

func newAudioFifo(rd io.Reader) *audioFifo {
	f := &audioFifo{
		fifo: make([]byte, 0),
		rd:   rd,
		pcm:  make([]byte, bytes20ms),
	}
	go f.run()
	return f
}

type audioFifo struct {
	mu   sync.Mutex
	rd   io.Reader
	fifo []byte

	filled int32
	closed int32
	pcm    []byte
}

func (a *audioFifo) Read20ms() ([]byte, error) {
	if atomic.LoadInt32(&a.closed) != 0 {
		return nil, io.EOF
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if (len(a.fifo) >= bytes20ms) && a.filled >= 0 {
		a.pcm, a.fifo = a.fifo[:bytes20ms], a.fifo[bytes20ms:]
		return a.pcm, nil
	}

	//log.Println("pcm fifo gone")
	atomic.StoreInt32(&a.filled, 0)
	for i := 0; i < bytes20ms; i++ {
		a.pcm[i] = 0
	}
	return a.pcm, nil
}

func (a *audioFifo) run() {
	rd := make([]byte, 10000)
	for {
		i, err := a.rd.Read(rd)
		if err != nil {
			log.Println("synth ws rd", err)
			atomic.StoreInt32(&a.closed, 1)
			return
		}

		a.mu.Lock()
		a.fifo = append(a.fifo, rd[:i]...)
		if len(a.fifo) >= filledFifo {
			//log.Println("pcm fifo ready")
			atomic.StoreInt32(&a.filled, 1)
		}
		a.mu.Unlock()
	}
}

func (m *RegularMedia) RunPcmTrack(ctx context.Context, stmid string, tid string, audio io.Reader) {
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

	m.room.AddSyntheticTrack(t)
	defer m.room.RemoveTrack(t)

	enc := opus.NewOpusEncoder()
	defer enc.Close()

	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()

	fifo := newAudioFifo(audio)
	for {
		select {
		case <-tick.C:
			buf, err := fifo.Read20ms()
			if err != nil {
				log.Println("fifo read err")
				return
			}
			// TODO: upscale here (x3), keeping last value from prev buffer
			encoded, err := enc.Encode(buf)
			if err != nil {
				log.Println("audio encoding error", err)
				return
			}

			if err := t.WriteSample(ms.Sample{Data: encoded, Duration: 20 * time.Millisecond}); err != nil {
				log.Println("synthetic write error", err)
				return
			}
			//log.Println("WriteSample")
		case <-ctx.Done():
			log.Println("file track ctx")
			return
		}
	}
}
