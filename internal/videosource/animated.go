package videosource

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const (
	addr    = "/tmp/sfu.sock"
	ramDisk = "/tmp"
	ftar    = "/opt/flexapix/flexatar/static3/server_saves/flexatars/dmisol1.p"

	typeFile = "file"
	typeMsg  = "message"

	animPayloadReady = "ready"
)

func NewAnimatedSource(ctx context.Context) (*AnimatedSource, error) {
	vs := &AnimatedSource{
		imgs: make(chan image.Image, 50),
	}
	var err error
	if vs.conn, err = net.Dial("unix", addr); err != nil {
		log.Println("dial", err)
		return nil, err
	}
	go vs.run(ctx)
	return vs, nil
}

type AnimatedSource struct {
	conn         net.Conn
	dir          string
	Delay        time.Duration
	t            time.Time
	imgs         chan image.Image
	canSendAudio int32
	index        int64
}

func (vs *AnimatedSource) run(ctx context.Context) {
	defer vs.conn.Close()

	dir, b, err := vs.mockJson()
	vs.dir = dir
	if err != nil {
		log.Println("mockJson failed: ", err)
		return
	}
	os.MkdirAll(dir, 0777)
	defer os.RemoveAll(dir)

	if _, err = vs.conn.Write(b); err != nil {
		log.Println("json sending", err)
		return
	}
	cntr := uint64(0)
	for {
		select {
		case <-ctx.Done():
			log.Println("animated source killed (ctx)")
			return
		default:
			b := make([]byte, 1024)
			i, err := vs.conn.Read(b)
			if err != nil {
				log.Println("sock rd", err)
				return
			}

			//log.Println("raw:", string(b))
			jsons := strings.Split(string(b[:i]), "\n")

			//log.Println("jsons:", jsons)
			for _, js := range jsons {
				if len(js) < 4 {
					break
				}

				ap := &AminPacket{}
				if err = json.Unmarshal([]byte(js), ap); err != nil {
					log.Println("unmarshal socket msg", err)
					return
				}

				switch ap.Type {
				case typeFile:
					name := ap.Payload
					//p.Println("name:", name)
					if err = vs.procImage(name); err != nil {
						log.Println("image decoding", name, err)
						return
					}
					cntr++
				case typeMsg:
					if ap.Payload == animPayloadReady {
						// trigger audio
						log.Println("READY msg, start processing audio")
						atomic.StoreInt32(&vs.canSendAudio, 1)
						continue
					} else {
						// TODO: log separately
						log.Println("ANIM ERR", ap.Payload)
						continue
					}
				default:
					log.Println("err unexpected type", js)
				}

			}
		}
	}

}

func (vs *AnimatedSource) Close() (err error) {
	log.Println("AnimatedSource close")
	return
}

func (vs *AnimatedSource) ID() (id string) {
	id = uuid.NewString()
	return
}

func (vs *AnimatedSource) Read() (image.Image, func(), error) {
	img := <-vs.imgs
	return img, func() {}, nil
}

func (vs *AnimatedSource) procImage(name string) error {
	r, err := os.Open(name)
	if err != nil {
		log.Println("jpeg read", err)
		return err
	}

	img, _, err := image.Decode(r)
	if err != nil {
		log.Println("jpeg decode", err)
		return err
	}

	vs.imgs <- img
	return nil
}
func (vs *AnimatedSource) WritePCM(pcm []byte) error {
	if len(vs.dir) == 0 {
		log.Println("thread not strted yet: no dir")
		return nil
	}
	if atomic.LoadInt32(&vs.canSendAudio) == 0 {
		return nil
	}
	seq := atomic.AddInt64(&vs.index, 1)
	name := fmt.Sprintf("%s/%08d.pcm", vs.dir, seq)
	if err := os.WriteFile(name, pcm, 0666); err != nil {
		log.Println("wr", err)
		return err
	}
	w := bufio.NewWriter(vs.conn)
	ts := time.Now().UnixMilli()

	ap := &AminPacket{
		Ts:      ts,
		Seq:     seq,
		Type:    typeFile,
		Payload: name,
	}

	b, err := json.Marshal(ap)
	if err != nil {
		log.Println("animpacket nmarshal", err)
		return err
	}
	if _, err = w.WriteString(string(b) + "\n"); err != nil {
		return err
	}
	w.Flush()
	return nil
}

type AminPacket struct {
	Ts      int64  `json:"ts"` // in ms
	Seq     int64  `json:"seq"`
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type InitialJson struct {
	Dir  string      `json:"dir"`
	Ftar interface{} `json:"ftar"`

	Static string `json:"static,omitempty"`

	W   int `json:"width,omitempty"`
	H   int `json:"height,omitempty"`
	FPS int `json:"fps,omitempty"`

	HeadPos interface{} `json:"head_position,omitempty"`
	Tattoo  interface{} `json:"tattoo,omitempty"`
	Bkg     int         `json:"video_bkg"`

	Glasses interface{} `json:"glasses,omitempty"`
	Merge   int         `json:"merge_type"`
	Color   interface{} `json:"color_filter,omitempty"`
	Pi      interface{} `json:"pattern_index,omitempty"`

	Enc string `json:"out_encoding,omitempty"`

	//Batch_s int  `json:"batch_size,omitempty"`
	//Blur    bool `json:"motion_blur,omitempty"`
	//Hat     bool `json:"hat,omitempty"`
	//VR      bool `json:"vr,omitempty"`
	//HairSeg bool `json:"hair_seg,omitempty"`
}

func (vs *AnimatedSource) mockJson() (string, []byte, error) {
	f, err := ioutil.ReadFile(path.Join("testdata", "init.json"))
	if err != nil {
		log.Println("init.json file read", err)
		return "", nil, err
	}
	ij := &InitialJson{}
	if err = json.Unmarshal(f, ij); err != nil {
		log.Println("init.json file unmarshal", err)
		return "", nil, err
	}
	ij.Dir = path.Join(ramDisk, uuid.NewString())
	ij.Ftar = ftar
	f, err = json.Marshal(ij)
	return ij.Dir, f, err
}
