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
	"strings"
	"sync"
	"time"

	"github.com/gabesullice/jq"
	"github.com/satori/go.uuid"
)

var (
	requestPushers = RequestPushers{pushers: make(map[string]*RequestPusher)}
)

type RequestPushers struct {
	sync.RWMutex
	pushers map[string]*RequestPusher
}

func (r *RequestPushers) add(id string, p *RequestPusher) {
	r.Lock()
	r.pushers[id] = p
	r.Unlock()
}

func (r *RequestPushers) get(id string) (*RequestPusher, bool) {
	r.RLock()
	p, ok := r.pushers[id]
	r.RUnlock()
	return p, ok
}

func (r *RequestPushers) remove(id string) {
	r.Lock()
	delete(r.pushers, id)
	r.Unlock()
}

type RequestPusher struct {
	sync.WaitGroup
	p http.Pusher
}

func (r *RequestPusher) Push(target string, opts *http.PushOptions) {
	log.Printf("Pushing: %s", target)
	r.Add(1)
	if err := r.p.Push(target, opts); err != nil {
		r.Done()
		log.Printf("Push Error: %s", err)
	}
	log.Printf("Pushed: %s", target)
}

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
	b.body.Reset()
}

func getPushLinks(ops map[string]jq.Op, b []byte) []string {
	var pushes []string
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
				pushes = append(pushes, url.RequestURI()+query)
			}
		}
	}
	return pushes
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
	http.Handle("/lib/", http.FileServer(http.Dir(".")))
	http.Handle("/dclient.html", http.FileServer(http.Dir(".")))
	http.Handle("/", handler(backend))
	log.Fatalln(http.ListenAndServeTLS(":443", "./server.crt", "./server.key", nil))
}

func handler(backend *httputil.ReverseProxy) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isPush := r.Header.Get("X-Push") == "ON"
		// Introduce some artificial latency
		if !isPush {
			time.Sleep(time.Millisecond * 150)
		}

		if r.Header.Get("Accept") != "application/vnd.api+json" {
			backend.ServeHTTP(w, r)
			return
		}

		rb := NewResponseBuffer(w)
		backend.ServeHTTP(rb, r)
		defer rb.Flush()

		requestID := r.Header.Get("X-Push-Request-ID")
		if requestID == "" {
			p, ok := w.(http.Pusher)
			if !ok {
				log.Println("HTTP/2 is not supported")
			} else {
				requestID = uuid.NewV4().String()
				requestPushers.add(requestID, &RequestPusher{p: p})
				defer requestPushers.remove(requestID)
			}
		}

		if requestID != "" {
			if pusher, ok := requestPushers.get(requestID); ok {
				ops := parsePushPlease(r.Header.Get("X-Push-Please"))
				if len(ops) > 0 {
					if bs, err := rb.ReadAll(); err == nil {
						if pushes := getPushLinks(ops, bs); len(pushes) > 0 {
							headers := r.Header
							headers.Set("X-Push", "ON")
							headers.Set("X-Push-Request-ID", requestID)
							opts := &http.PushOptions{Header: headers}
							for _, target := range pushes {
								pusher.Push(target, opts)
							}
						}
					}
				}

				if isPush {
					pusher.Done()
				} else {
					pusher.Wait()
				}
			}
		}
	})
}

func parsePushPlease(please string) map[string]jq.Op {
	ops := make(map[string]jq.Op)
	for _, part := range strings.Split(please, ";") {
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
	return ops
}
