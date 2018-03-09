package server

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

type responseBuffer struct {
	body *bytes.Buffer
	w    http.ResponseWriter
}

func newResponseBuffer(w http.ResponseWriter) responseBuffer {
	var buf bytes.Buffer
	return responseBuffer{
		body: &buf,
		w:    w,
	}
}

func (b responseBuffer) Header() http.Header {
	return b.w.Header()
}

func (b responseBuffer) WriteHeader(h int) {
	b.w.WriteHeader(h)
}

func (b responseBuffer) Write(bs []byte) (int, error) {
	return b.body.Write(bs)
}

func (b *responseBuffer) ReadAll() ([]byte, error) {
	body := b.body.Bytes()
	return ioutil.ReadAll(bytes.NewReader(body))
}

func (b *responseBuffer) Flush() {
	io.Copy(b.w, bytes.NewReader(b.body.Bytes()))
	b.body.Reset()
}
