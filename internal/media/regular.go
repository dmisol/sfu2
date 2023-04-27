package media

import (
	"context"
	"io"
	"log"
	"path"
	"sync/atomic"

	"github.com/dmisol/sfu2/internal/defs"
	"github.com/dmisol/sfu2/internal/media/opus"
	"github.com/dmisol/sfu2/internal/videosource"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
)

const (
	audiochan = 1
	opusRate  = 48000

	runDummy = true
)

// NewRegularMedia() optionally interfaces audio to bot and creates either a or a+v tracks from bot's responce
func NewRegularMedia(room defs.Room, bot io.ReadWriter, ftar string) defs.Media {
	m := &RegularMedia{
		room: room,
	}
	if bot != nil {
		log.Println("using bot")
		m.xGress = bot
		m.useBot = 1
		m.ftar = ftar
	}
	return m
}

type RegularMedia struct {
	room   defs.Room
	useBot int32

	ftar   string
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

	if atomic.LoadInt32(&m.useBot) > 0 {

		dec = opus.NewOpusDecoder(3)
		defer dec.Close()

		if runDummy {
			stmid := uuid.NewString()
			// just for debugging
			go m.RunPcmFileTrack(ctx, stmid)
			vs, err := videosource.NewDummySource(path.Join("testdata", "img.jpeg"))
			if err != nil {
				log.Println("dummyVideoSource", err)
			} else {
				go m.RunH264Track(ctx, stmid, vs)
			}
		}

		stmid := uuid.NewString()
		go m.RunPcmTrack(ctx, stmid, m.xGress, m.ftar)
	}

	for {
		p, _, err := t.ReadRTP()
		if err != nil {
			log.Println("packet read error", err)
			return
		}
		if len(p.Payload) == 0 {
			log.Println("rx empty rtp, skipping")
			continue
		}

		if atomic.LoadInt32(&m.useBot) > 0 {
			pcm16k, err := dec.Decode(p.Payload)

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
