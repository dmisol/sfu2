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
	audiochan = 1
	opusRate  = 48000
)

func NewOpusDecoder(il int) *OpusDecoder {
	e := C.int(0)
	er := &e
	d := &OpusDecoder{
		dec:        C.opus_decoder_create(C.int(opusRate), C.int(audiochan), er),
		interleave: il,
	}
	return d
}

type OpusDecoder struct {
	dec        *C.OpusDecoder
	interleave int
}

// Dec() is a debugging/auxiliary function to return linear pcm samples at original sampling rate
func (d *OpusDecoder) Dec(payload []byte) ([]int16, error) {
	samplesPerFrame := int(C.opus_packet_get_samples_per_frame((*C.uchar)(&payload[0]), C.int(48000)))

	pcm := make([]int16, samplesPerFrame)
	samples := C.opus_decode(d.dec, (*C.uchar)(&payload[0]), C.opus_int32(len(payload)), (*C.opus_int16)(&pcm[0]), C.int(cap(pcm)/audiochan), 0)
	if samples < 0 {
		log.Println("opus decoding failed")
		return nil, errors.New("opus decoding failed")
	}
	return pcm, nil
}

func (d *OpusDecoder) Decode(payload []byte) ([]byte, error) {
	pcm, err := d.Dec(payload)
	if err != nil {
		return nil, err
	}

	out := make([]byte, 0)
	outBuffer := bytes.NewBuffer(out)
	i := 0
	for _, v := range pcm {
		if i == 0 {
			if err = binary.Write(outBuffer, binary.LittleEndian, v); err != nil {
				return nil, err
			}
		}
		i++
		if i >= d.interleave {
			i = 0
		}
	}

	return outBuffer.Bytes(), nil
}

func (d *OpusDecoder) Close() error {
	C.opus_decoder_destroy(d.dec)
	return nil
}
