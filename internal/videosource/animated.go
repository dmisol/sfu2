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
	debug            = false
)

func NewAnimatedSource(ctx context.Context) (*AnimatedSource, error) {
	id := uuid.NewString()
	vs := &AnimatedSource{
		imgs: make(chan image.Image, 50),
		id:   id,
		dir:  path.Join(ramDisk, id),
	}
	var err error
	if vs.conn, err = net.Dial("unix", addr); err != nil {
		vs.Println("dial", err)
		return nil, err
	}

	if err = os.MkdirAll(vs.dir, 0777); err != nil {
		vs.Println("dir creating", err)
		return nil, err
	}
	vs.Println("dir created", vs.dir, err)

	go vs.run(ctx)
	return vs, nil
}

type AnimatedSource struct {
	conn  net.Conn
	dir   string
	Delay time.Duration

	imgs         chan image.Image
	canSendAudio int32
	index        int64

	id string

	sec         int32
	fps         int32
	pkts, bytes int32
}

func (vs *AnimatedSource) run(ctx context.Context) {
	defer vs.Println("DONE")
	defer vs.conn.Close()
	if !debug {
		defer os.RemoveAll(vs.dir)
	}

	b, err := vs.mockJson()
	if err != nil {
		vs.Println("mockJson failed: ", err)
		return
	}

	if _, err = vs.conn.Write(b); err != nil {
		vs.Println("json sending", err)
		return
	}
	vs.Println("starting")
	for {
		select {
		case <-ctx.Done():
			vs.Println("animated source killed (ctx)")
			return
		default:
			b := make([]byte, 1024)
			i, err := vs.conn.Read(b)
			if err != nil {
				vs.Println("sock rd", err)
				return
			}

			vs.Println("raw:", string(b))
			jsons := strings.Split(string(b[:i]), "\n")

			//vs.Println("jsons:", jsons)
			for _, js := range jsons {
				if len(js) < 4 {
					break
				}

				ap := &AminPacket{}
				if err = json.Unmarshal([]byte(js), ap); err != nil {
					vs.Println("unmarshal socket msg", err)
					return
				}

				switch ap.Type {
				case typeFile:
					name := ap.Payload
					// vs.Println("name:", name)
					if err = vs.procImage(name); err != nil {
						vs.Println("image decoding", name, err)
						return
					}
					atomic.AddInt32(&vs.fps, 1)
					if !debug {
						os.Remove(name)
					}
				case typeMsg:
					if ap.Payload == animPayloadReady {
						// trigger audio
						vs.Println("READY msg, start processing audio")
						atomic.StoreInt32(&vs.canSendAudio, 1)
						continue
					} else {
						// TODO: log separately
						vs.Println("ANIM ERR", ap.Payload)
						continue
					}
				default:
					vs.Println("err unexpected type", js)
				}

			}
		}
	}

}

func (vs *AnimatedSource) Close() (err error) {
	vs.Println("AnimatedSource close")
	return
}

func (vs *AnimatedSource) ID() (id string) {
	return vs.id
}

func (vs *AnimatedSource) Read() (image.Image, func(), error) {
	img := <-vs.imgs
	return img, func() {}, nil
}

func (vs *AnimatedSource) procImage(name string) error {
	r, err := os.Open(name)
	if err != nil {
		vs.Println("jpeg read", err)
		return err
	}

	img, _, err := image.Decode(r)
	if err != nil {
		vs.Println("jpeg decode", err)
		return err
	}

	vs.imgs <- img
	return nil
}
func (vs *AnimatedSource) WritePCM(pcm []byte) error {

	if debug {
		atomic.AddInt32(&vs.pkts, 1)
		atomic.AddInt32(&vs.bytes, int32(len(pcm)))
		sec := int32(time.Now().Second())
		if sec != atomic.LoadInt32(&vs.sec) {
			vs.Println("audio", atomic.LoadInt32(&vs.pkts), atomic.LoadInt32(&vs.bytes),
				"fps", atomic.LoadInt32(&vs.fps))
			atomic.StoreInt32(&vs.fps, 0)
			atomic.StoreInt32(&vs.pkts, 0)
			atomic.StoreInt32(&vs.bytes, 0)
			atomic.StoreInt32(&vs.sec, sec)
		}
	}

	if len(vs.dir) == 0 {
		vs.Println("thread not strted yet: no dir")
		return nil
	}
	if atomic.LoadInt32(&vs.canSendAudio) == 0 {
		return nil
	}
	seq := atomic.AddInt64(&vs.index, 1)
	name := fmt.Sprintf("%s/%08d.pcm", vs.dir, seq)
	// vs.Println("writing", name)

	if err := os.WriteFile(name, pcm, 0666); err != nil {
		vs.Println("wr", err)
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
		vs.Println("animpacket nmarshal", err)
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

func (vs *AnimatedSource) mockJson() ([]byte, error) {
	f, err := ioutil.ReadFile(path.Join("testdata", "init.json"))
	if err != nil {
		vs.Println("init.json file read", err)
		return nil, err
	}
	ij := &InitialJson{}
	if err = json.Unmarshal(f, ij); err != nil {
		vs.Println("init.json file unmarshal", err)
		return nil, err
	}
	ij.Dir = vs.dir
	ij.Ftar = []string{ftar}
	f, err = json.Marshal(ij)
	return f, err
}

func (vs *AnimatedSource) Println(i ...interface{}) {
	log.Println("AnimSrc", vs.id, i)
}
