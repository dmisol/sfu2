package media

// #cgo linux CFLAGS: -I/usr/include/opus
// #cgo linux LDFLAGS: -L/usr/lib/x86_64-linux-gnu -lopus
// #include <opus.h>
import "C"

import (
	"io"
	"log"

	"github.com/pion/webrtc/v3"
)

func (m *RegularMedia) RunPcmTrack(stmid string, tid string, audio io.Reader) {
	// 1. create trackLocalStaticRTP for bot's responses

	// TODO: force ptime ?
	t, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{
			MimeType:     "audio/opus",
			ClockRate:    48000,
			Channels:     2,
			SDPFmtpLine:  "ptime=20", //"minptime=10;useinbandfec=1",
			RTCPFeedback: nil,
		}, tid, stmid)
	if err != nil {
		log.Println("local track creating", err)
		return
	}

	// 2. m.room.AddSyntheticTrack()
	m.room.AddSyntheticTrack(t)
	defer m.room.RemoveTrack(t)

	// 3. encoder
	var cerror C.int
	enc := C.opus_encoder_create(
		C.opus_int32(opusRate),
		C.int(audiochan),
		C.OPUS_APPLICATION_VOIP,
		&cerror,
	)
	defer C.opus_encoder_destroy(enc)
	for {
		// TODO: []byte -> []int16
		/*
			// 4. { read, encode, write to trackLocal }

			bs := make([]byte, 960) // 20ms!
			i, err := audio.Read(bs)
			if i != len(bs) || err != nil {
				log.Println("got from pcm ingress", i, err)
			}

			// see https://github.com/pion/mediadevices/blob/master/pkg/codec/opus/opus.go
			var n C.opus_int32
			encoded := make([]byte, 1024)
			n = C.opus_encode(
				enc,
				(*C.opus_int16)(&bs[0]),
				C.int(len(bs)/2),
				(*C.uchar)(&encoded[0]),
				C.opus_int32(cap(encoded)),
			)
			if n < 0 {
				log.Println("encoding error")
			}
			t.Write(encoded[:n:n])
		*/
	}
}
