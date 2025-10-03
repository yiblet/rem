package tui

import (
	"bufio"
	"io"
)

// Pager provides buffered reading with seek support for streaming content display
type Pager struct {
	inner io.ReadSeeker
	buf   *bufio.Reader
}

// NewPager creates a new Pager wrapping the given ReadSeeker
func NewPager(r io.ReadSeeker) *Pager {
	return &Pager{
		inner: r,
		buf:   bufio.NewReader(r),
	}
}

// Seek implements seeking by repositioning underlying reader and resetting buffer
func (p *Pager) Seek(offset int64, whence int) (int64, error) {
	pos, err := p.inner.Seek(offset, whence)
	if err != nil {
		return pos, err
	}
	p.buf.Reset(p.inner)
	return pos, nil
}

// ReadLine reads a line from the pager (up to and including newline)
func (p *Pager) ReadLine() (string, error) {
	return p.buf.ReadString('\n')
}

// ReadRune reads a single rune (needed for regex matching)
func (p *Pager) ReadRune() (rune, int, error) {
	return p.buf.ReadRune()
}

// Read implements io.Reader interface
func (p *Pager) Read(b []byte) (int, error) {
	return p.buf.Read(b)
}

// Close closes the underlying reader if it implements io.Closer
func (p *Pager) Close() error {
	if closer, ok := p.inner.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}