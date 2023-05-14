package media

import (
	"context"
	"log"
	"os"
	"path"
	"time"

	"github.com/dmisol/sfu2/internal/defs"
	"github.com/dmisol/sfu2/internal/media/opus"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	ms "github.com/pion/webrtc/v3/pkg/media"
)

func runPcmFileTrack(ctx context.Context, room defs.Room, stmid string) {
	tid := uuid.NewString()

	b, err := os.ReadFile(path.Join("testdata", "48k.raw"))
	if err != nil {
		log.Println("TrPcmFile file read", err)
		return
	}

	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()

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
		log.Println("TrPcmFile local track creating", err)
		return
	}

	// 2. m.room.AddSyntheticTrack()
	room.AddSyntheticTrack(t, nil)
	defer room.RemoveTrack(t)

	// 3. encoder
	enc := opus.NewOpusEncoder()
	defer enc.Close()

	ind := 0
	var x []byte
	for {
		select {
		case <-tick.C:
			if ind+bytes20ms < len(b) {
				x = b[ind : ind+bytes20ms]
				ind += bytes20ms
			} else {
				ind = 0
				x = b[:bytes20ms]
			}
			encoded, err := enc.Encode(x)
			if err != nil {
				log.Println("TrPcmFile audio encoding error", err)
				return
			}

			if err := t.WriteSample(ms.Sample{Data: encoded, Duration: 20 * time.Millisecond}); err != nil {
				log.Println("TrPcmFile synthetic write error", err)
				return
			}
		case <-ctx.Done():
			log.Println("TrPcmFile file track ctx")
			return
		}
	}
}
