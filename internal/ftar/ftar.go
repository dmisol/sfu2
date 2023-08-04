package ftar

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

const (
	pngCntr = 3
	iconTag = "PreviewImage" //"firstImage"
)

var (
	jpegSignature = []byte{0xff, 0xd8, 0xff}
	pngSignature  = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
)

type jsonTag struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

func NewFtar(name string) (*Ftar, error) {
	f := &Ftar{
		File: name,
		Date: time.Now(),
	}
	f.Key()
	if err := f.fetchPng(); err != nil {
		return nil, err
	}
	return f, nil
}

type Ftar struct {
	File string
	Id   string // self name from Icon
	Icon string

	Fixed bool
	Date  time.Time
}

func (f *Ftar) Key() {
	s := path.Base(f.File)
	id := strings.Split(s, ".")[0]

	f.Id = id
	return
}

func (f *Ftar) fetchPng() error {
	// TODO:
	return fmt.Errorf("failed to fetch icon")
}

func (f *Ftar) CopyAndFetchPng() error {
	in, err := os.Open(f.File)
	if err != nil {
		return err
	}
	defer in.Close()

	name := path.Base(f.File)
	full := path.Join(FixedOut, name)
	out, err := os.Create(full)
	if err != nil {
		return err
	}
	defer out.Close()
	f.File = full // file available for animation engine

	head := make([]byte, 8)

	var jt *jsonTag

	gotIcon := false
	cntr := 0

	for {
		n, err := in.Read(head)
		if err == io.EOF && n == 0 && gotIcon {
			return nil
		}
		if err != nil && n != 8 {
			log.Println("tag read failed", n, err)
			return err
		}
		if _, err = out.Write(head); err != nil {
			log.Println("tag write failed", err)
			return err
		}

		sz := 0
		for i := 0; i < 7; i++ {
			sz += int(head[i]) << (i << 3)
		}
		//log.Println("sz is", sz)

		b := make([]byte, sz)
		n, err = in.Read(b)
		if err == io.EOF && n == 0 && gotIcon {
			return nil
		}
		if err != nil && n != sz {
			log.Println("block read failed", n, err)
			return err
		}
		jtn, ext := f.blockType(cntr, b)
		if jtn == nil {
			if jt == nil {
				return errors.New("initial block is not json")
			}
			// log.Println(jt.Type, jt.Index, sz, ext)

			if jt.Type == iconTag && len(ext) != 0 {
				f.Id += ext
				f.Icon = path.Join(IconPath, f.Id)

				err := ioutil.WriteFile(f.Icon, b, 0666)
				if err != nil {
					log.Println("icon failed", err)
					return err
				}
				gotIcon = true
			}
		} else {
			jt = jtn
		}

		if _, err = out.Write(b); err != nil {
			log.Println("block write failed", err)
			return err
		}

		cntr++
	}
}

func (f *Ftar) blockType(cntr int, b []byte) (*jsonTag, string) {
	if strings.HasPrefix(string(b), string(jpegSignature)) {
		return nil, ".jpeg"
	}
	if strings.HasPrefix(string(b), string(pngSignature)) {
		return nil, ".png"
	}
	if cntr%2 == 0 && strings.HasPrefix(string(b), "{") {
		jt := &jsonTag{}
		if err := json.Unmarshal(b, jt); err != nil {
			return nil, ""
		}
		return jt, string(b)
	}
	return nil, ""
}
