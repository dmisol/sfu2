package bot

import (
	"context"
	"errors"
	"log"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

var errNotConected = errors.New("bot not connected")

const (
	wsRun int32 = iota
	wsClosed
)

func NewBot(ctx context.Context, url string) *Bot {
	b := &Bot{}

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		b.Println("bot start failed", err)
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
}

func (b *Bot) run(ctx context.Context) {
	defer b.ws.Close()

	<-ctx.Done()
	b.Println("bot done")
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
	if atomic.LoadInt32(&b.wsOk) == wsRun {
		err := b.ws.WriteMessage(websocket.BinaryMessage, pcm)
		if err != nil {
			atomic.StoreInt32(&b.wsOk, wsClosed)
			b.Println("bot egress error", err)
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
			b.Println("bot rd", err)
			return 0, err
		}
		copy(pcm, buf)
		return len(buf), nil
	}
	b.Println("bot not connected")
	return 0, errNotConected
}

func (b *Bot) Println(i ...interface{}) {
	log.Println("BOT", i)
}
