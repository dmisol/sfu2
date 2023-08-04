package media

import (
	"context"
	"log"
	"path"

	"github.com/dmisol/sfu2/internal/defs"
	"github.com/dmisol/sfu2/internal/videosource"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
)

func NewDebugMedia(room defs.Room) defs.Media {
	m := &DebugMedia{
		room: room,
	}
	return m
}

type DebugMedia struct {
	room defs.Room
}

func (m *DebugMedia) OnAudioTrack(ctx context.Context, t *webrtc.TrackRemote) {
	stmid := uuid.NewString()
	// just for debugging
	go runPcmFileTrack(ctx, m.room, stmid)
	if defs.DebugVideo {
		vs, err := videosource.NewDummySource(path.Join("testdata", "img.jpeg"))
		if err != nil {
			m.Println("dummyVideoSource", err)
		} else {
			go runH264Track(ctx, m.room, stmid, vs)
		}
	}
}

func (m *DebugMedia) OnVideoTrack(_ context.Context, t *webrtc.TrackRemote) {}

func (m *DebugMedia) Println(i ...interface{}) {
	log.Println("DM", i)
}
