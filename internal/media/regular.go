package media

import (
	"io"
	"log"

	"github.com/dmisol/sfu2/internal/defs"
	"github.com/pion/webrtc/v3"
)

func NewRegularMedia(room defs.Room, bot io.WriteCloser) defs.Media {
	m := &RegularMedia{
		room: room,
		bot:  bot,
	}
	return m
}

type RegularMedia struct {
	room defs.Room
	bot  io.WriteCloser
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

	buf := make([]byte, 1500)
	for {
		i, _, err := t.Read(buf)
		if err != nil {
			return
		}

		if m.bot != nil {
			if _, err = m.bot.Write(buf[:i]); err != nil {
				log.Println("bot write error")
				m.bot.Close()
				m.bot = nil
			}
		}

		if _, err = trackLocal.Write(buf[:i]); err != nil {
			return
		}
	}
}
