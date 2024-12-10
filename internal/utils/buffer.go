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

// NewBufferCloser returns a buffer that wraps bytes.Buffer and satisfies
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

// Write provides a nop implementation for io.discardCloser.
func (b discardCloser) Write(p []byte) (int, error) {
	return len(p), nil
}

// Read provides a nop implementation for io.discardCloser.
func (b discardCloser) Read(p []byte) (int, error) {
	return len(p), nil
}

// Close provides a nop implementation for io.discardCloser.
func (b discardCloser) Close() error {
	return nil
}

// String implements fmt.Stringer for io.discardCloser.
func (b discardCloser) String() string {
	return ""
}
