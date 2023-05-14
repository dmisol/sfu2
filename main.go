package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net"
	"net/http"
	"path"
	"text/template"

	"github.com/dmisol/sfu2/internal/bot"
	"github.com/dmisol/sfu2/internal/defs"
	"github.com/dmisol/sfu2/internal/media"
	"github.com/dmisol/sfu2/internal/rtc"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

var (
	addr          = flag.String("addr", ":8080", "http service address")
	indexTemplate = &template.Template{}

	roomBot, roomGeneric *rtc.Room
	conf                 *defs.Conf
)

type websocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

func main() {
	var err error
	conf, err = defs.ReadConf("conf.yaml")
	if err != nil {
		log.Fatal(err)
	}

	roomGeneric = rtc.NewRoom("ChatRoom")
	roomBot = rtc.NewRoom("BotRoom")
	// websocket handler
	http.HandleFunc("/ws", roomHandler)

	// index.html handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join("static", "index.html"))
	})

	http.HandleFunc("/view", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join("static", "view.html"))
	})

	if len(conf.Hosts) == 0 {
		log.Fatal(http.ListenAndServe(*addr, nil))
	} else {
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist("https://flexatar.com", "flexatar.com"),
			Cache:      autocert.DirCache("/tmp/certs"),
		}

		cfg := &tls.Config{
			GetCertificate: m.GetCertificate,
			NextProtos: []string{
				"http/1.1", acme.ALPNProto,
			},
		}

		// Let's Encrypt tls-alpn-01 only works on port 443.
		ln, err := net.Listen("tcp4", "0.0.0.0:443")
		if err != nil {
			panic(err)
		}

		lnTls := tls.NewListener(ln, cfg)
		panic(http.Serve(lnTls, nil))

	}
}

// Handle incoming websockets
func roomHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), rtc.Timeout)
	defer cancel()

	runBot := r.URL.Query().Has("bot")
	ftar := r.URL.Query().Get("ftar")

	log.Println(ftar, runBot)
	if runBot {
		log.Println("user with animated bot, bot room")
		aiBot := bot.NewBot(ctx, conf.BotUrl) // to enambe bot act as a peer
		media := media.NewRegularMedia(roomBot, aiBot, ftar)
		rtc.NewUser(ctx, roomBot, conf, media, w, r)
		return
	}

	if len(ftar) > 0 {
		log.Println("user with flexatar, generic room")
		media := media.NewAnimatedHumanMedia(roomGeneric, ftar)
		rtc.NewUser(ctx, roomGeneric, conf, media, w, r)
		return
	}
	media := media.NewRegularMedia(roomGeneric, nil, "")
	rtc.NewUser(ctx, roomGeneric, conf, media, w, r)

}
