package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/rocketlaunchr/https-go"
)

var (
	listenFlag = flag.String("listen", "127.0.0.1:6774", "HTTP listen address")
	httpsFlag  = flag.Bool("https", false, "enable HTTPS with self-signed certificate")
)

func main() {
	flag.Parse()

	cors := handlers.CORS(handlers.AllowedOrigins([]string{"*"}))
	handler := cors(http.DefaultServeMux)
	handler = handlers.LoggingHandler(os.Stderr, handler)

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
