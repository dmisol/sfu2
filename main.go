package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"path"
	"strings"
	"syscall"

	"github.com/dmisol/sfu2/internal/bot"
	"github.com/dmisol/sfu2/internal/defs"
	ftarStorage "github.com/dmisol/sfu2/internal/ftar"
	"github.com/dmisol/sfu2/internal/media"
	"github.com/dmisol/sfu2/internal/rtc"
	chi "github.com/go-chi/chi/v5"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

const (
	useDebugRoom     = true
	unixProxyData    = "/tmp/processing.sock"
	unixProxyAndroid = "/tmp/android.sock"
	ct               = "Content-Type"
)

var (
	addr = flag.String("addr", ":8080", "http service address")

	roomBot, roomGeneric, roomDebug *rtc.Room
	conf                            *defs.Conf

	storage *ftarStorage.Cache
)

func main() {
	syscall.Umask(0)

	var err error
	conf, err = defs.ReadConf("conf.yaml")
	if err != nil {
		log.Fatal(err)
	}

	api, err := rtc.NewApi(true, true, 3478)
	if err != nil {
		log.Println("sfu starting", err)
		return
	}
	roomGeneric = rtc.NewRoom("ChatRoom", api)
	roomBot = rtc.NewRoom("BotRoom", api)

	if useDebugRoom {

		debugApi, err := rtc.NewApi(true, true /*defs.DebugVideo*/, 3480)
		if err != nil {
			log.Println("debugRoom starting", err)
			return
		}

		roomDebug = rtc.NewRoom("DebugRoom", debugApi)
		media := media.NewDebugMedia(roomDebug)
		media.OnAudioTrack(context.Background(), nil)
	}

	storage = ftarStorage.NewCache()

	c := chi.NewRouter()

	// websocket handler
	c.Get("/ws", roomHandler)

	// TODO: auth
	// forward request for processing
	c.Get("/data", proxyHandler)
	c.Post("/data", proxyHandler)
	c.Get("/android", proxyHandler)
	c.Post("/android", proxyHandler)

	// trigger to involve newly-created ftars
	c.Post("/ftar", func(w http.ResponseWriter, r *http.Request) {
		fn, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println("/ftar POST body", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err = storage.AddFtar(string(fn)); err != nil {
			log.Println("/ftar add", string(fn), err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	})

	// commands and files
	c.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join("static", "index.html"))
	})
	c.Get("/{cmd}", func(w http.ResponseWriter, r *http.Request) {
		fn := chi.URLParam(r, "cmd")
		log.Println("/cmd, fn=", fn)
		if strings.Contains(fn, ".") {
			http.ServeFile(w, r, path.Join("static", fn))
			return
		}
		http.ServeFile(w, r, fmt.Sprintf("static/%s.html", fn))
	})
	c.Get("/file/*", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		log.Println("/file:, fn=", p)
		if strings.Contains(p, "..") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		http.ServeFile(w, r, path.Join("static", p))
	})

	c.Get("/icon/list", func(w http.ResponseWriter, r *http.Request) {
		group := r.URL.Query().Get("group")
		b, err := storage.GetList(group)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})
	c.Get("/icon/{img}", func(w http.ResponseWriter, r *http.Request) {
		fn := chi.URLParam(r, "img")
		log.Println("/icon", fn)
		http.ServeFile(w, r, path.Join(ftarStorage.IconPath, fn))
	})

	if len(conf.Hosts) == 0 {
		log.Fatal(http.ListenAndServe(*addr, c))
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
		panic(http.Serve(lnTls, c))
	}
}

// Handle incoming websockets
func roomHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), rtc.Timeout)
	defer cancel()

	runBot := r.URL.Query().Has("bot")
	ftarWeb := r.URL.Query().Get("ftar")
	deb := r.URL.Query().Has("deb")

	var ftar string
	var err error

	if len(ftarWeb) > 0 {
		ftar, err = storage.ResolveFtar(ftarWeb)
		if err != nil {
			log.Println("unexpected ftar", ftarWeb, err)
			ftar = ""
		}
	}

	log.Println(ftar, runBot, deb)

	if deb && useDebugRoom {
		log.Println("debug room")
		media := media.NewDebugMedia(roomDebug) // just any: will not me used
		rtc.NewUser(ctx, roomDebug, conf, media, w, r)
		return
	}

	if runBot {
		log.Println("user with animated bot, bot room")
		aiBot := bot.NewBot(ctx, conf.BotUrl) // to enambe bot act as a peer
		media := media.NewRegularMedia(roomBot, aiBot, ftar, cancel)
		rtc.NewUser(ctx, roomBot, conf, media, w, r)
		return
	}

	if len(ftar) > 0 {
		log.Println("user with flexatar, generic room")
		media := media.NewAnimatedHumanMedia(roomGeneric, ftar, cancel)
		rtc.NewUser(ctx, roomGeneric, conf, media, w, r)
		return
	}
	log.Println("regular user (no flexatar), generic room")
	media := media.NewRegularMedia(roomGeneric, nil, "", cancel)
	rtc.NewUser(ctx, roomGeneric, conf, media, w, r)

}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	m := r.Method
	log.Println("serving", r.URL.Path, m)
	url := "http://tmp.1"

	us := unixProxyData
	if strings.Contains(r.URL.Path, "android") {
		us = unixProxyAndroid
	}

	values := r.URL.Query()
	cntr := 0
	for k, v := range values {
		if cntr == 0 {
			url += fmt.Sprintf("?%s=%s", k, v[0])
		} else {
			url += fmt.Sprintf("&%s=%s", k, v[0])
		}
		cntr++
	}

	cl := &http.Client{}

	if len(conf.Proxy) > 0 {
		url = conf.Proxy
		cl = http.DefaultClient
	} else {
		cl = &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", us)
				},
			},
		}
	}

	req, err := http.NewRequest(m, url, r.Body)

	if err != nil {
		log.Println("req error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	hdr := r.Header.Get(ct)
	if len(hdr) > 0 {
		req.Header.Set(ct, hdr)
	}

	resp, err := cl.Do(req)
	if err != nil {
		log.Println("do error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("body error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set(ct, resp.Header.Get(ct))
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}
