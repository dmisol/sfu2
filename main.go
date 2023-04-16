package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net"
	"net/http"
	"path"
	"text/template"

	"github.com/dmisol/sfu2/internal/defs"
	"github.com/dmisol/sfu2/internal/rtc"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

var (
	addr          = flag.String("addr", ":8080", "http service address")
	indexTemplate = &template.Template{}
)

type websocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

func main() {
	c, err := defs.ReadConf("conf.yaml")
	if err != nil {
		log.Fatal(err)
	}

	room := rtc.NewRoom()
	// websocket handler
	http.HandleFunc("/ws", room.WebsocketHandler)

	// index.html handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join("static", "index.html"))
	})

	http.HandleFunc("/view", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join("static", "view.html"))
	})

	if len(c.Hosts) == 0 {
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
