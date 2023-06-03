package media

import (
	"log"
	"sync/atomic"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type Bootstrap struct {
	NeedPli int32
	bs      []*rtp.Packet
	track   *webrtc.TrackLocalStaticRTP
	seq     uint16
}

func (b *Bootstrap) Write(pkts []*rtp.Packet) error {
	if atomic.LoadInt32(&b.NeedPli) > 0 {
		b.Println("bootstrap wanted")
	}

	if b.hasBootstrap(pkts) {
		b.bs = pkts
		b.Println("bootstrap updated")
		err := b.sendFrame(pkts)
		atomic.StoreInt32(&b.NeedPli, 0)
		return err
	}
	if atomic.LoadInt32(&b.NeedPli) > 0 {
		ts := pkts[0].Timestamp
		for _, p := range b.bs {
			p.Timestamp = ts
		}
		err := b.sendFrame(b.bs)
		atomic.StoreInt32(&b.NeedPli, 0)
		return err
	}
	return b.sendFrame(pkts)
}

func (b *Bootstrap) hasBootstrap(pkts []*rtp.Packet) bool {
	if !pkts[len(pkts)-1].Marker {
		b.Println("no marker at the end")
		return false
	}
	for _, p := range pkts {
		if b.isH264Keyframe(p.Payload) {
			return true
		}
	}
	return false
}

func (b *Bootstrap) sendFrame(pkts []*rtp.Packet) error {
	for _, p := range pkts {
		p.SequenceNumber = b.seq
		if err := b.track.WriteRTP(p); err != nil {
			b.Println("writing to track", err)
			return err
		}
		b.seq++
	}
	return nil
}

func (b *Bootstrap) isH264Keyframe(payload []byte) bool {
	if len(payload) < 1 {
		return false
	}
	nalu := payload[0] & 0x1F
	if nalu == 0 {
		// reserved
		return false
	} else if nalu <= 23 {
		// simple NALU
		return nalu == 5
	} else if nalu == 24 || nalu == 25 || nalu == 26 || nalu == 27 {
		// STAP-A, STAP-B, MTAP16 or MTAP24
		i := 1
		if nalu == 25 || nalu == 26 || nalu == 27 {
			// skip DON
			i += 2
		}
		for i < len(payload) {
			if i+2 > len(payload) {
				return false
			}
			length := uint16(payload[i])<<8 |
				uint16(payload[i+1])
			i += 2
			if i+int(length) > len(payload) {
				return false
			}
			offset := 0
			if nalu == 26 {
				offset = 3
			} else if nalu == 27 {
				offset = 4
			}
			if offset >= int(length) {
				return false
			}
			n := payload[i+offset] & 0x1F
			if n == 7 {
				return true
			} else if n >= 24 {
				// is this legal?
				b.Println("Non-simple NALU within a STAP")
			}
			i += int(length)
		}
		if i == len(payload) {
			return false
		}
		return false
	} else if nalu == 28 || nalu == 29 {
		// FU-A or FU-B
		if len(payload) < 2 {
			return false
		}
		if (payload[1] & 0x80) == 0 {
			// not a starting fragment
			return false
		}
		return payload[1]&0x1F == 7
	}
	return false
}

func (b *Bootstrap) Println(i ...interface{}) {
	log.Println("BS", b.track.StreamID(), i)
}
