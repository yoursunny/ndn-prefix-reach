package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/rocketlaunchr/https-go"
)

var (
	listenFlag = flag.String("listen", "127.0.0.1:6774", "HTTP listen address")
	httpsFlag  = flag.Bool("https", false, "enable HTTPS with self-signed certificate")
)

var handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RemoteAddr, r.URL)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.DefaultServeMux.ServeHTTP(w, r)
})

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()

	if *httpsFlag {
		server, e := https.Server("", https.GenerateOptions{Host: "localhost"})
		if e != nil {
			log.Fatalln(e)
		}
		server.Addr = *listenFlag
		server.Handler = handler
		log.Fatalln(server.ListenAndServeTLS("", ""))
	} else {
		log.Fatalln(http.ListenAndServe(*listenFlag, handler))
	}
}

func init() {
	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)
		w.Write([]byte("User-Agent: *\nDisallow: /\n"))
	})
}
