package media

import (
	"context"
	"io"
	"log"
	"sync/atomic"

	"github.com/dmisol/sfu2/internal/defs"
	"github.com/dmisol/sfu2/internal/media/opus"
	"github.com/google/uuid"
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

func (m *RegularMedia) OnVideoTrack(_ context.Context, t *webrtc.TrackRemote) {
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

func (m *RegularMedia) OnAudioTrack(ctx context.Context, t *webrtc.TrackRemote) {
	trackLocal := m.room.AddTrack(t)
	defer m.room.RemoveTrack(trackLocal)

	var dec *opus.OpusDecoder

	/*
		// debugging!
		var enc *opus.OpusEncoder
		var dec2 *opus.OpusDecoder
	*/

	if atomic.LoadInt32(&m.useBot) > 0 {

		dec = opus.NewOpusDecoder(3)
		defer dec.Close()

		/*
			// debugging!
			enc = opus.NewOpusEncoder()
			defer enc.Close()
			dec2 = opus.NewOpusDecoder(3)
			defer dec2.Close()
		*/

		sid := uuid.NewString()
		tid := uuid.NewString()

		//go m.RunPcmFileTrack(ctx, sid, tid, m.xGress)
		go m.RunPcmTrack(ctx, sid, tid, m.xGress)
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
			pcm16k, err := dec.Decode(p.Payload)

			/*
				pcm48k, err := dec.Dec(p.Payload)
				if err != nil {
					log.Println("decoding", err)
					atomic.StoreInt32(&m.useBot, 0)
				}
				b, err := enc.Enc(pcm48k)
				if err != nil {
					log.Println("encoding", err)
					atomic.StoreInt32(&m.useBot, 0)
				}
				//log.Println(len(p.Payload), len(b))
				pcm16k, err := dec2.Decode(b)
			*/

			/*
				bs48k, err := dec2.Decode(p.Payload)
				if err != nil {
					log.Println("decoding", err)
					atomic.StoreInt32(&m.useBot, 0)
				}
				//fmt.Printf("dec bytes: %d %2x %2x %2x %2x \t", len(bs48k), bs48k[0], bs48k[1], bs48k[2], bs48k[3])
				b, err := enc.Encode(bs48k)
				if err != nil {
					log.Println("encoding", err)
					atomic.StoreInt32(&m.useBot, 0)
				}
				pcm16k, err := dec2.Decode(b) // 48k infact
			*/

			if err != nil {
				log.Println("decoding", err)
				atomic.StoreInt32(&m.useBot, 0)
			} else {
				_, err := m.xGress.Write(pcm16k)
				if err != nil {
					log.Println("bot write error")
					atomic.StoreInt32(&m.useBot, 0)
				}
			}
		}

		if err = trackLocal.WriteRTP(p); err != nil {
			log.Println("track local write error", err)
			return
		}
	}
}
