package bot

import (
	"context"
	"errors"
	"log"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var errNotConected = errors.New("bot not connected")

const (
	botUrl = "ws://localhost:8081/ws"

	wsRun int32 = iota
	wsClosed
)

func NewBot(ctx context.Context) *Bot {
	b := &Bot{}

	c, _, err := websocket.DefaultDialer.Dial(botUrl, nil)
	if err != nil {
		log.Println("bot start failed", err)
		b.wsOk = wsClosed
		return b
	}
	b.ws = c
	b.wsOk = wsRun
	go b.run(ctx)
	return b
}

type Bot struct {
	ws   *websocket.Conn // to send audio
	wsOk int32

	// f *os.File

	bytesOut int
	bytesIn  int32

	t time.Time
	/*
	   // testing endorer!
	   mu  sync.Mutex
	   tst [][]byte
	*/
}

func (b *Bot) run(ctx context.Context) {
	defer b.ws.Close()

	<-ctx.Done()
	log.Println("bot done")
}

func (b *Bot) Close() error {
	if atomic.LoadInt32(&b.wsOk) == wsRun {
		atomic.StoreInt32(&b.wsOk, wsClosed)
		if b.ws != nil {
			b.ws.Close()
		}
	}
	return nil
}

func (b *Bot) Write(pcm []byte) (int, error) {

	t := time.Now()
	if t.Second() != b.t.Second() {

		bi := atomic.LoadInt32(&b.bytesIn)
		atomic.AddInt32(&b.bytesIn, -bi)

		log.Println("bot egress:", b.bytesOut, "ingress", bi)
		b.t = t
		b.bytesOut = 0
	}

	b.bytesOut += len(pcm)

	if atomic.LoadInt32(&b.wsOk) == wsRun {
		err := b.ws.WriteMessage(websocket.BinaryMessage, pcm)
		if err != nil {
			atomic.StoreInt32(&b.wsOk, wsClosed)
			log.Println("bot egress error", err)
		}

		return len(pcm), err
	}
	return 0, errNotConected
}

func (b *Bot) Read(pcm []byte) (int, error) {
	if atomic.LoadInt32(&b.wsOk) == wsRun {
		_, buf, err := b.ws.ReadMessage()
		if err != nil {
			atomic.StoreInt32(&b.wsOk, wsClosed)
			log.Println("bot rd", err)
			return 0, err
		}
		copy(pcm, buf)
		atomic.AddInt32(&b.bytesIn, int32(len(buf)))
		return len(buf), nil
	}
	log.Println("bot not connected")
	return 0, errNotConected
}
