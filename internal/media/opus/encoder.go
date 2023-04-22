package opus

// #cgo linux CFLAGS: -I/usr/include/opus
// #cgo linux LDFLAGS: -L/usr/lib/x86_64-linux-gnu -lopus
// #include <opus.h>
import "C"
import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
)

const (
	bytes20ms = 1920
)

func NewOpusEncoder() *OpusEncoder {
	var cerror C.int
	enc := C.opus_encoder_create(
		C.opus_int32(opusRate),
		C.int(audiochan),
		C.OPUS_APPLICATION_VOIP,
		&cerror,
	)
	oe := &OpusEncoder{enc: enc}
	return oe
}

type OpusEncoder struct {
	enc *C.OpusEncoder
}

func (e *OpusEncoder) Encode(bs []byte) ([]byte, error) {
	in := bs
	var pcm []int16
	var sample int16
	var x []byte
	for len(pcm) < len(bs)/2 {
		x, in = in[:2], in[2:]
		buf := bytes.NewReader(x)
		err := binary.Read(buf, binary.LittleEndian, &sample)
		if err != nil {
			log.Println("encoding, binary", err)
			return nil, err
		}
		pcm = append(pcm, sample)
	}
	return e.Enc(pcm)
}

func (e *OpusEncoder) Enc(pcm []int16) ([]byte, error) {
	var n C.opus_int32
	encoded := make([]byte, 2000)
	n = C.opus_encode(
		e.enc,
		(*C.opus_int16)(&pcm[0]),
		C.int(len(pcm)),
		(*C.uchar)(&encoded[0]),
		C.opus_int32(cap(encoded)),
	)
	if n < 0 {
		return nil, errors.New("encoding error")
	}
	return encoded[:n], nil
}

func (e *OpusEncoder) Close() error {
	C.opus_encoder_destroy(e.enc)
	return nil
}
