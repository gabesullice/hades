package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	hades "github.com/gabesullice/hades/lib/server"
)

func main() {
	url, err := url.Parse(os.Args[1])
	if err != nil {
		log.Fatalln("Could not parse backend URL")
	}
	log.Printf("Started proxy for %v", url)
	backend := httputil.NewSingleHostReverseProxy(url)
	d := backend.Director
	backend.Director = func(r *http.Request) {
		r.Header.Set("X-Forwarded-Proto", "https")
		d(r)
	}
	http.Handle("/", hades.NewProxy(backend))
	log.Fatalln(http.ListenAndServeTLS(":443", "./server.crt", "./server.key", nil))
}
