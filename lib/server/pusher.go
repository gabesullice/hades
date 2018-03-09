package server

import (
	"log"
	"net/http"
	"sync"
)

var ()

type pusher struct {
	sync.WaitGroup
	p http.Pusher
}

func (r *pusher) push(target string, opts *http.PushOptions) {
	log.Printf("Pushing: %s", target)
	r.Add(1)
	if err := r.p.Push(target, opts); err != nil {
		r.Done()
		log.Printf("Push Error: %s", err)
	}
	log.Printf("Pushed: %s", target)
}

type pusherMap struct {
	sync.RWMutex
	pushers map[string]*pusher
}

func (r *pusherMap) add(id string, p *pusher) {
	r.Lock()
	r.pushers[id] = p
	r.Unlock()
}

func (r *pusherMap) get(id string) (*pusher, bool) {
	r.RLock()
	p, ok := r.pushers[id]
	r.RUnlock()
	return p, ok
}

func (r *pusherMap) remove(id string) {
	r.Lock()
	delete(r.pushers, id)
	r.Unlock()
}
