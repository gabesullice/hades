package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/savaki/jq"
)

type PushProcessor struct {
	ops []jq.Op
	rw  http.ResponseWriter
}

func (p PushProcessor) Header() http.Header {
	return p.rw.Header()
}

func (p PushProcessor) WriteHeader(h int) {
	p.rw.WriteHeader(h)
}

func (p PushProcessor) Write(b []byte) (int, error) {
	for _, op := range p.ops {
		if data, err := op.Apply(b); err != nil {
			log.Println(err)
		} else {
			log.Printf("%s\n", data)
		}
	}
	return p.rw.Write(b)
}

func main() {
	url, _ := url.Parse(os.Args[1])
	log.Printf("Started proxy for %v", url)
	backend := httputil.NewSingleHostReverseProxy(url)
	backend.Director = chainDirectors(
		//logRequest,
		//logHeader,
		changeHost(backend.Director, url),
	)
	log.Fatalln(http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ops []jq.Op
		for _, path := range os.Args[2:] {
			ops = append(ops, jq.Must(jq.Parse(path)))
		}
		backend.ServeHTTP(PushProcessor{
			ops: ops,
			rw:  w,
		}, r)
	})))
}

func chainDirectors(dirs ...func(*http.Request)) func(*http.Request) {
	return func(r *http.Request) {
		for k, _ := range dirs {
			dirs[k](r)
		}
	}
}

func changeHost(d func(*http.Request), url *url.URL) func(*http.Request) {
	return func(r *http.Request) {
		path := r.URL.Path
		d(r)
		r.Host = url.Host
		r.URL = url
		r.URL.Path = path
	}
}

func logRequest(r *http.Request) {
	dump, _ := httputil.DumpRequest(r, false)
	log.Printf("%s", dump)
}

func logHeader(r *http.Request) {
	log.Printf("%+v", r.Header["X-Push-Request"])
}
