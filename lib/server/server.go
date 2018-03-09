package server

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gabesullice/jq"
	"github.com/satori/go.uuid"
)

func NewProxy(backend *httputil.ReverseProxy) http.Handler {
	return &handler{
		backend: backend,
		pushers: pusherMap{pushers: make(map[string]*pusher)},
	}
}

type handler struct {
	pushers pusherMap
	backend *httputil.ReverseProxy
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	isPush := r.Header.Get("X-Push") == "ON"

	if r.Header.Get("Accept") != "application/vnd.api+json" {
		h.backend.ServeHTTP(w, r)
		return
	}

	rb := newResponseBuffer(w)
	h.backend.ServeHTTP(rb, r)
	defer rb.Flush()

	requestID := r.Header.Get("X-Push-Request-ID")
	if requestID == "" {
		p, ok := w.(http.Pusher)
		if !ok {
			log.Println("HTTP/2 is not supported")
		} else {
			requestID = uuid.NewV4().String()
			h.pushers.add(requestID, &pusher{p: p})
			defer h.pushers.remove(requestID)
		}
	}

	if requestID != "" {
		if pusher, ok := h.pushers.get(requestID); ok {
			ops := parsePushPlease(r.Header.Get("X-Push-Please"))
			if len(ops) > 0 {
				if bs, err := rb.ReadAll(); err == nil {
					if pushes := getPushLinks(ops, bs); len(pushes) > 0 {
						headers := r.Header
						headers.Set("X-Push", "ON")
						headers.Set("X-Push-Request-ID", requestID)
						opts := &http.PushOptions{Header: headers}
						for _, target := range pushes {
							pusher.push(target, opts)
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
