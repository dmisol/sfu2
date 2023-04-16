package bot

import (
	"context"
	"log"

	"github.com/dmisol/sfu2/internal/defs"
	"github.com/gorilla/websocket"
)

func NewBot(room defs.Room) *Bot {
	b := &Bot{room: room}
	return b
}

type Bot struct {
	room defs.Room
	ws   *websocket.Conn // to send audio
}

func (b *Bot) Run(ctx context.Context) {
	// TODO:
	// 1. Create fake peer connection
	// 2. publish fake TrackLocalStarticRtp to play audio from "AI"
	// 3. establish ws to "AI" to send and receive audio
	<-ctx.Done()
	log.Println("bot done")
}

func (b *Bot) Close() error {
	return nil
}

func (b *Bot) Write([]byte) (int, error) {
	return 0, nil
}
