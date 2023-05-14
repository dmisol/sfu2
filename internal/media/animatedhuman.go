package media

import (
	"context"
	"log"

	"github.com/dmisol/sfu2/internal/defs"
	"github.com/dmisol/sfu2/internal/media/opus"
	"github.com/dmisol/sfu2/internal/videosource"
	"github.com/pion/webrtc/v3"
)

func NewAnimatedHumanMedia(room defs.Room, ftar string) defs.Media {
	m := &AnimatedHumanMedia{
		room: room,
		ftar: ftar,
		pcm:  make(chan []byte, 250), // 5 sec
	}
	return m
}

type AnimatedHumanMedia struct {
	room defs.Room
	ftar string
	pcm  chan []byte
}

func (m *AnimatedHumanMedia) OnVideoTrack(_ context.Context, t *webrtc.TrackRemote) {
	m.Println("video tack from user not needed")
}

func (m *AnimatedHumanMedia) OnAudioTrack(ctx context.Context, t *webrtc.TrackRemote) {
	trackLocal := m.room.AddTrack(t)
	defer m.room.RemoveTrack(trackLocal)

	dec := opus.NewOpusDecoder(1)
	defer dec.Close()

	stmid := t.StreamID()
	m.Println("strating video with flexatar", m.ftar)
	as, err := videosource.NewAnimatedSource(ctx)
	if err != nil {
		m.Println("can't start video track", err)
		return
	}
	go runH264Track(ctx, m.room, stmid, as)

	for {
		p, _, err := t.ReadRTP()
		if err != nil {
			m.Println("packet read error", err)
			return
		}
		if len(p.Payload) == 0 {
			m.Println("rx empty rtp, skipping")
			continue
		}

		if err = trackLocal.WriteRTP(p); err != nil {
			m.Println("track local write error", err)
			return
		}

		pcm48k, err := dec.Decode(p.Payload)
		if err != nil {
			m.Println("anim decodinng", err)
			return
		}
		if as != nil {
			if err := as.WritePCM(pcm48k); err != nil {
				m.Println("anim write", err)
				return
			}
		}
	}
}

func (m *AnimatedHumanMedia) Println(i ...interface{}) {
	log.Println("AHM", i)
}
