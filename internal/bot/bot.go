package bot

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func NewBot() *Bot {
	b := &Bot{}

	log.Println("decoder created")
	// testing now: write to file
	f, err := os.Create(path.Join("/tmp", fmt.Sprintf("%s.pcm", uuid.NewString())))
	if err != nil {
		log.Println("test file", err)
		return nil
	}
	b.f = f

	return b
}

type Bot struct {
	ws *websocket.Conn // to send audio

	f *os.File

	bytes   int
	packets int
	t       time.Time
}

func (b *Bot) Run(ctx context.Context) {
	u := url.URL{Scheme: "ws", Host: "localhost:8081", Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println("bot start failed", err)
		return
	}
	defer c.Close()
	b.ws = c

	<-ctx.Done()
	log.Println("bot done")
}

func (b *Bot) Close() error {
	if b.f != nil {
		b.f.Close()
	}
	if b.ws != nil {
		b.ws.Close()
	}
	return nil
}

func (b *Bot) Write(pcm []byte) (int, error) {
	t := time.Now()
	if t.Second() != b.t.Second() {
		log.Println(b.bytes, b.packets)
		b.t = t
		b.bytes = 0
		b.packets = 0
	}

	b.packets++
	b.bytes += len(pcm)

	if b.f != nil {
		b.f.Write(pcm)
	}

	if b.ws != nil {
		err := b.ws.WriteMessage(websocket.BinaryMessage, pcm)
		if err != nil {
			log.Println("bot egress", err)
		}

		return len(pcm), err

	}
	return 0, nil
}

func (b *Bot) Read(pcm []byte) (int, error) {
	return 0, nil
}
