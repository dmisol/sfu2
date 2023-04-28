package media

import (
	"context"
	"log"
	"math/rand"

	"github.com/google/uuid"
	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/x264"
	"github.com/pion/webrtc/v3"
)

func (m *RegularMedia) RunH264Track(ctx context.Context, stmid string, vs mediadevices.VideoSource) {

	x264Params, err := x264.NewParams()
	if err != nil {
		log.Println("x264Params", err)
	}
	x264Params.Preset = x264.PresetMedium
	x264Params.BitRate = 1_000_000 // 1mbps
	x264Params.KeyFrameInterval = 30
	// log.Println("x264Params")

	codecSelector := mediadevices.NewCodecSelector(
		mediadevices.WithVideoEncoders(&x264Params),
	)
	// log.Println("codecSelector")

	vt := mediadevices.NewVideoTrack(vs, codecSelector)

	track, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		uuid.NewString(),
		stmid)
	if err != nil {
		log.Println("video track failed", err)
		return
	}

	bs := &Bootstrap{
		track: track,
	}

	// TODO: RTCPReader() ?
	m.room.AddSyntheticTrack(track, &bs.NeedPli)
	defer m.room.RemoveTrack(track)

	rr, err := vt.NewRTPReader(x264Params.RTPCodec().MimeType, rand.Uint32(), 1000)
	if err != nil {
		log.Println("NewRtpReader", err)
		return
	}

	go func() {
		<-ctx.Done()
		log.Println("animator ctopped ctx")
		m.room.RemoveTrack(track)
		rr.Close()
	}()

	for {
		pkts, _, err := rr.Read()
		if err != nil {
			log.Println("mediadevices rd", err)
			return
		}
		if err = bs.Write(pkts); err != nil {
			log.Println("h264 video done", err)
			return
		}
	}
}
