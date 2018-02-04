package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"

	"github.com/gabesullice/jq"
)

var (
	reArray = regexp.MustCompile(`^\s*\[\s*(\d+)(\s*:\s*(\d+))?\s*]\s*$`)
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
		data, err := op.Apply(b)
		if err != nil {
			log.Println("Apply Error:", err)
		}

		rw, ok := p.rw.(http.Pusher)
		if !ok {
			log.Println("HTTP/2 is not supported")
		}

		var links []string
		if err := json.Unmarshal(data, &links); err != nil {
			log.Println("Unmarshal Error:", err)
		}

		for _, link := range links {
			if url, err := url.Parse(link); err == nil {
				if err := rw.Push(url.Path, nil); err == nil {
					log.Printf("Pushed: %s", url.Path)
				}
			}
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
	http.Handle("/static/", http.FileServer(http.Dir(".")))
	http.Handle("/", handler(backend))
	log.Fatalln(http.ListenAndServeTLS(":443", "./server.crt", "./server.key", nil))
}

func handler(backend *httputil.ReverseProxy) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ops []jq.Op
		for _, path := range r.Header["X-Push-Request"] {
			ops = append(ops, jq.Must(jq.Parse(path)))
		}
		if _, ok := w.(http.Pusher); !ok {
			log.Println("HTTP/2 is not supported by the client.")
		}
		backend.ServeHTTP(PushProcessor{
			ops: ops,
			rw:  w,
		}, r)
	})
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
