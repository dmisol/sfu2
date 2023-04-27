package videosource

import (
	"image"
	_ "image/jpeg"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pion/mediadevices"
)

const dummyInterval = 100 * time.Millisecond // 10fps to start with

func NewDummySource(name string) (mediadevices.VideoSource, error) {
	r, err := os.Open(name)
	if err != nil {
		log.Println("image read", err)
		return nil, err
	}

	var img image.Image
	if img, _, err = image.Decode(r); err != nil {
		log.Println("image decode", err)
		return nil, err
	}

	ds := &dummySource{
		img: img,
	}
	return ds, nil
}

type dummySource struct {
	img image.Image
	t   time.Time
}

func (vs *dummySource) Close() (err error) {
	log.Println("dummySource close")
	return
}

func (vs *dummySource) ID() (id string) {
	id = uuid.NewString()
	return
}

func (vs *dummySource) Read() (image.Image, func(), error) {
	now := time.Now()
	toFire := vs.t.Add(dummyInterval)
	if toFire.After(now) {
		toSleep := toFire.Sub(now)
		//log.Println("sleeping", toSleep)
		time.Sleep(toSleep)
	}
	vs.t = time.Now()
	return vs.img, func() {}, nil
}
