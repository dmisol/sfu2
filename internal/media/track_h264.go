package media

import (
	"context"
	"log"
	"math/rand"

	"github.com/dmisol/sfu2/internal/defs"
	"github.com/google/uuid"
	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/x264"
	"github.com/pion/webrtc/v3"
)

func runH264Track(ctx context.Context, room defs.Room, stmid string, vs mediadevices.VideoSource) {

	x264Params, err := x264.NewParams()
	if err != nil {
		log.Println("TrH264 x264Params", err)
	}
	x264Params.Preset = x264.PresetMedium
	x264Params.BitRate = 1_000_000 // 1mbps
	x264Params.KeyFrameInterval = 30

	codecSelector := mediadevices.NewCodecSelector(
		mediadevices.WithVideoEncoders(&x264Params),
	)

	vt := mediadevices.NewVideoTrack(vs, codecSelector)

	track, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		uuid.NewString(),
		stmid)
	if err != nil {
		log.Println("TrH264 video track failed", err)
		return
	}

	bs := &Bootstrap{
		track: track,
	}

	room.AddSyntheticTrack(track, &bs.NeedPli)
	defer room.RemoveTrack(track)

	rr, err := vt.NewRTPReader(x264Params.RTPCodec().MimeType, rand.Uint32(), 1000)
	if err != nil {
		log.Println("TrH264 NewRtpReader", err)
		return
	}

	go func() {
		<-ctx.Done()
		log.Println("TrH264 animator ctopped ctx")
		room.RemoveTrack(track)
		rr.Close()
	}()

	for {
		pkts, _, err := rr.Read()
		if err != nil {
			log.Println("TrH264 mediadevices rd", err)
			return
		}
		if err = bs.Write(pkts); err != nil {
			log.Println("TrH264 h264 video done", err)
			return
		}
	}
}
