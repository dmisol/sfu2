package media

// #cgo linux CFLAGS: -I/usr/include/opus
// #cgo linux LDFLAGS: -L/usr/lib/x86_64-linux-gnu -lopus
// #include <opus.h>
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"sync/atomic"

	"github.com/dmisol/sfu2/internal/defs"
	"github.com/pion/webrtc/v3"
)

const (
	audiochan = 1
	opusRate  = 48000
)

func NewRegularMedia(room defs.Room, bot io.ReadWriter) defs.Media {
	m := &RegularMedia{
		room: room,
	}
	if bot != nil {
		log.Println("using bot")
		m.xGress = bot
		m.useBot = 1
	}
	return m
}

type RegularMedia struct {
	room   defs.Room
	useBot int32

	xGress io.ReadWriter
}

func (m *RegularMedia) OnVideoTrack(t *webrtc.TrackRemote) {
	trackLocal := m.room.AddTrack(t)
	defer m.room.RemoveTrack(trackLocal)

	buf := make([]byte, 1500)
	for {
		i, _, err := t.Read(buf)
		if err != nil {
			return
		}

		if _, err = trackLocal.Write(buf[:i]); err != nil {
			return
		}
	}
}

func (m *RegularMedia) OnAudioTrack(t *webrtc.TrackRemote) {
	trackLocal := m.room.AddTrack(t)
	defer m.room.RemoveTrack(trackLocal)

	var dec *C.OpusDecoder

	if atomic.LoadInt32(&m.useBot) > 0 {
		e := C.int(0)
		er := &e
		dec = C.opus_decoder_create(C.int(opusRate), C.int(audiochan), er)
		defer C.opus_decoder_destroy(dec)

		sid := fmt.Sprintf("bot_stm_%s", t.StreamID())
		tid := fmt.Sprintf("bot_audio_%s", t.ID())
		go m.RunPcmTrack(sid, tid, m.xGress)
	}

	for {
		p, _, err := t.ReadRTP()
		if err != nil {
			log.Println("packet read error", err)
			return
		}
		//log.Println("rtp", p, len(p.Payload))
		if len(p.Payload) == 0 {
			log.Println("rx empty rtp, skipping")
			continue
		}

		if atomic.LoadInt32(&m.useBot) > 0 {
			samplesPerFrame := int(C.opus_packet_get_samples_per_frame((*C.uchar)(&p.Payload[0]), C.int(48000)))

			pcm := make([]int16, samplesPerFrame)
			samples := C.opus_decode(dec, (*C.uchar)(&p.Payload[0]), C.opus_int32(len(p.Payload)), (*C.opus_int16)(&pcm[0]), C.int(cap(pcm)/audiochan), 0)
			if samples < 0 {
				log.Println("opus decoding failed")
				return
			}

			pcmData := make([]byte, 0)
			pcmBuffer := bytes.NewBuffer(pcmData)
			for _, v := range pcm {
				binary.Write(pcmBuffer, binary.LittleEndian, v)
			}

			_, err := m.xGress.Write(pcmBuffer.Bytes())
			if err != nil {
				log.Println("bot write error")
				atomic.StoreInt32(&m.useBot, 0)
			}
		}

		if err = trackLocal.WriteRTP(p); err != nil {
			log.Println("track local write error", err)
			return
		}
	}
}
