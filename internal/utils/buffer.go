package utils

import (
	"bytes"
	"fmt"
	"io"
	"sync"
)

// Buffer interface satisfies io.ReadCloser, io.WriteCloser, and fmt.Stringer.
type Buffer interface {
	fmt.Stringer
	io.Reader
	io.Writer
	io.Closer
}

type bufferCloser struct {
	buf *bytes.Buffer
	mux *sync.RWMutex
}

// BufferCloser returns a buffer that wraps bytes.Buffer and satisfies
// io.Closer.
func NewBufferCloser() Buffer {
	return bufferCloser{
		buf: &bytes.Buffer{},
		mux: &sync.RWMutex{},
	}
}

func (b bufferCloser) Write(p []byte) (int, error) {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.buf.Write(p)
}

func (b bufferCloser) Read(p []byte) (int, error) {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.buf.Read(p)
}

func (b bufferCloser) Close() error {
	return nil
}

func (b bufferCloser) String() string {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.buf.String()
}

type discardCloser struct{}

// NewDiscardCloser returns a buffer that simply discards everything.
func NewDiscardCloser() Buffer {
	return discardCloser{}
}

func (b discardCloser) Write(p []byte) (int, error) {
	return len(p), nil
}

func (b discardCloser) Read(p []byte) (int, error) {
	return len(p), nil
}

func (b discardCloser) Close() error {
	return nil
}

func (b discardCloser) String() string {
	return ""
}
