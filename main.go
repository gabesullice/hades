package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gabesullice/jq"
)

var (
	reArray = regexp.MustCompile(`^\s*\[\s*(\d+)(\s*:\s*(\d+))?\s*]\s*$`)
)

type ResponseBuffer struct {
	body *bytes.Buffer
	w    http.ResponseWriter
}

func NewResponseBuffer(w http.ResponseWriter) ResponseBuffer {
	var buf bytes.Buffer
	return ResponseBuffer{
		body: &buf,
		w:    w,
	}
}

func (b ResponseBuffer) Header() http.Header {
	return b.w.Header()
}

func (b ResponseBuffer) WriteHeader(h int) {
	b.w.WriteHeader(h)
}

func (b ResponseBuffer) Write(bs []byte) (int, error) {
	return b.body.Write(bs)
}

func (b *ResponseBuffer) ReadAll() ([]byte, error) {
	body := b.body.Bytes()
	return ioutil.ReadAll(bytes.NewReader(body))
}

func (b *ResponseBuffer) Flush() {
	io.Copy(b.w, bytes.NewReader(b.body.Bytes()))
}

func processPushes(ops map[string]jq.Op, b []byte, p http.Pusher) {
	for part, op := range ops {
		data, err := op.Apply(b)
		if err != nil {
			continue
		}

		trimmed := strings.TrimSpace(part)
		var query string
		if strings.Contains(trimmed, "?") {
			query = "?" + strings.Split(trimmed, "?")[1]
		} else {
			query = ""
		}

		var links []string
		if err := json.Unmarshal(data, &links); err != nil {
			var link string
			if err := json.Unmarshal(data, &link); err != nil {
				continue
			}
			links = append(links, link)
		}

		for _, link := range links {
			if url, err := url.Parse(link); err == nil {
				push := url.Path + query
				log.Printf("Pushing: %s", push)
				opts := &http.PushOptions{
					Header: http.Header{"X-Push": []string{"ON"}},
				}
				if err := p.Push(push, opts); err == nil {
					log.Printf("Pushed: %s", push)
				} else {
					log.Printf("Push Error: %s", err)
				}
			}
		}
	}
}

func main() {
	url, _ := url.Parse(os.Args[1])
	log.Printf("Started proxy for %v", url)
	backend := httputil.NewSingleHostReverseProxy(url)
	d := backend.Director
	backend.Director = func(r *http.Request) {
		r.Header.Set("X-Forwarded-Proto", "https")
		d(r)
	}
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.Handle("/jsonapi/", handler(backend))
	log.Fatalln(http.ListenAndServeTLS(":443", "./server.crt", "./server.key", nil))
}

func handler(backend *httputil.ReverseProxy) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(http.Pusher); !ok {
			log.Println("HTTP/2 is not supported by the client.")
		}
		rb := NewResponseBuffer(w)
		if r.Header.Get("X-Push") != "ON" {
			time.Sleep(time.Millisecond * 150)
		}
		backend.ServeHTTP(rb, r)
		p, ok := w.(http.Pusher)
		if !ok {
			log.Println("HTTP/2 is not supported")
		}
		ops := parsePaths(r.Header["X-Push-Please"])
		if len(ops) > 0 {
			if bs, err := rb.ReadAll(); err == nil {
				processPushes(ops, bs, p)
			}
		}
		rb.Flush()
	})
}

func parsePaths(headers []string) map[string]jq.Op {
	ops := make(map[string]jq.Op)
	for _, headerValues := range headers {
		for _, part := range strings.Split(headerValues, ";") {
			// Invalid paths are simply ignored.
			trimmed := strings.TrimSpace(part)
			var path string
			if strings.Contains(trimmed, "?") {
				path = strings.Split(trimmed, "?")[0]
			} else {
				path = trimmed
			}
			if op, err := jq.Parse(path); err == nil {
				ops[trimmed] = op
			}
		}
	}
	return ops
}
