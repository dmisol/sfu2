package bot

import (
	"context"
	"log"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

func NewBot() *Bot {
	b := &Bot{}

	/*
		// testing now: write to file
		f, err := os.Create(path.Join("/tmp", fmt.Sprintf("%s.pcm", uuid.NewString())))
		if err != nil {
			log.Println("test file", err)
			return nil
		}
		b.f = f
	*/
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

func (b *Bot) Run(ctx context.Context) {
	u := url.URL{Scheme: "ws", Host: "localhost:8081", Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println("bot start failed", err)
		return
	}
	defer c.Close()
	b.ws = c
	atomic.StoreInt32(&b.wsOk, 1)

	<-ctx.Done()
	log.Println("bot done")
}

func (b *Bot) Close() error {
	if atomic.LoadInt32(&b.wsOk) == 1 {
		atomic.StoreInt32(&b.wsOk, 0)
		// b.f.Close()
		if b.ws != nil {
			b.ws.Close()
		}
	}
	return nil
}

func (b *Bot) Write(pcm []byte) (int, error) {
	/*
		b.mu.Lock()
		b.tst = append(b.tst, pcm)
		b.mu.Unlock()
	*/

	t := time.Now()
	if t.Second() != b.t.Second() {

		bi := atomic.LoadInt32(&b.bytesIn)
		atomic.AddInt32(&b.bytesIn, -bi)

		log.Println("bot egress:", b.bytesOut, "ingress", bi)
		b.t = t
		b.bytesOut = 0
	}

	b.bytesOut += len(pcm)
	/*
		if b.f != nil {
			_, _ = b.f.Write(pcm)
		}
	*/
	if atomic.LoadInt32(&b.wsOk) == 1 {
		err := b.ws.WriteMessage(websocket.BinaryMessage, pcm)
		if err != nil {
			atomic.StoreInt32(&b.wsOk, 0)
			log.Println("bot egress error", err)
		}

		return len(pcm), err

	}
	return 0, nil
}

func (b *Bot) Read(pcm []byte) (int, error) {
	/*
		b.mu.Lock()
		defer b.mu.Unlock()

		if len(b.tst) > 0 {
			pcm, b.tst = b.tst[0], b.tst[1:]
			return 1920, nil
		}
		return 0, nil
	*/
	if atomic.LoadInt32(&b.wsOk) == 1 {
		_, buf, err := b.ws.ReadMessage()
		if err != nil {
			atomic.StoreInt32(&b.wsOk, 0)
			log.Println("ingress error", err)
			return 0, err
		}
		//log.Println("ingress", mt, len(buf))
		pcm = buf
		atomic.AddInt32(&b.bytesIn, int32(len(buf)))
		return len(buf), nil
	}
	return 0, nil
}
