package media

import (
	"context"
	"fmt"
	"log"

	"github.com/dmisol/sfu2/internal/defs"
	"github.com/dmisol/sfu2/internal/media/opus"
	"github.com/dmisol/sfu2/internal/videosource"
	"github.com/google/uuid"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

const two_video_tracks = true

func NewAnimatedHumanMedia(room defs.Room, ftar string, cancel context.CancelFunc) defs.Media {
	m := &AnimatedHumanMedia{
		cancel: cancel,
		room:   room,
		ftar:   ftar,
		pcm:    make(chan []byte, 250), // 5 sec
	}
	return m
}

type AnimatedHumanMedia struct {
	cancel context.CancelFunc
	room   defs.Room
	ftar   string
	pcm    chan []byte
}

func (m *AnimatedHumanMedia) OnVideoTrack(_ context.Context, t *webrtc.TrackRemote) {
	if two_video_tracks {
		// create stream with prefix "cam-" derived from t.StreamID()
		// explisively collect bootstrap and add video track as synthetic track

		stmid := fmt.Sprintf("cam-%s", t.StreamID())
		track, err := webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
			"cam-"+uuid.NewString(),
			stmid)
		if err != nil {
			m.Println("preserving original video track failed", err)
			return
		}

		bs := &Bootstrap{
			track: track,
		}

		m.room.AddSyntheticTrack(track, &bs.NeedPli)
		defer m.room.RemoveTrack(track)
		var frame []*rtp.Packet
		for {
			p, _, err := t.ReadRTP()
			if err != nil {
				m.Println("cam read", err)
				return
			}

			// collect keyframe to respond pli
			frame = append(frame, p)

			if p.Marker {
				// m.Println("frame", len(frame))
				err = bs.Write(frame)
				if err != nil {
					m.Println("cam write", err)
					return
				}
				frame = frame[:0]
			} /*else {
				m.Println("still collecting", len(frame))
			}*/

		}
		/*
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
			} */

	} else {
		m.Println("video tack from user not used")
	}
}

func (m *AnimatedHumanMedia) OnAudioTrack(ctx context.Context, t *webrtc.TrackRemote) {
	defer func() {
		m.Println("onAudioTrack done")
		m.cancel()
	}()
	trackLocal := m.room.AddTrack(t)
	defer m.room.RemoveTrack(trackLocal)

	dec := opus.NewOpusDecoder(1)
	defer dec.Close()

	stmid := t.StreamID()
	m.Println("strating video with flexatar", m.ftar)
	as, err := videosource.NewAnimatedSource(ctx, m.ftar)
	if err != nil {
		m.Println("can't start video track", err)
		m.Println("canceling ctx for serving user")
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
	log.Println("AnimHmn", i)
}
