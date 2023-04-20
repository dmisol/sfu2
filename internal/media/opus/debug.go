package opus

import (
	"encoding/gob"
	"fmt"
	"os"
	"path"

	"github.com/google/uuid"
)

func NewMixWriter(name string) (*MixWriter, error) {
	mw := &MixWriter{}
	mn := name
	if mn == "" {
		mn = fmt.Sprintf("%s.gob", uuid.NewString())
	}
	f, err := os.Create(path.Join("/tmp", mn))
	if err != nil {
		return nil, err
	}
	mw.f = f
	mw.enc = gob.NewEncoder(f)
	return mw, nil
}

type MixWriter struct {
	f   *os.File
	enc *gob.Encoder
}

func (m *MixWriter) Write(p []byte) (int, error) {
	err := m.enc.Encode(len(p))
	if err != nil {
		return 0, err
	}
	err = m.enc.Encode(p)
	return len(p), err
}

func (m *MixWriter) Close() error {
	m.f.Close()
	return nil
}

func NewMixReader(name string) (*MixReader, error) {
	m := &MixReader{}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	m.f = f
	m.dec = gob.NewDecoder(f)
	return m, nil
}

func (m *MixReader) Read() ([]byte, error) {
	var l int
	err := m.dec.Decode(&l)
	if err != nil {
		return nil, err
	}
	b := make([]byte, l)
	err = m.dec.Decode(&b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

type MixReader struct {
	f   *os.File
	dec *gob.Decoder
}
