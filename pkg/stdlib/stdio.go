package stdlib

import (
	"bufio"
	"io"
	"strings"
)

// Stdio is a program's standard streams. Stdout is buffered; reading
// stdin or writing stderr flushes it first, so output stays in order.
type Stdio struct {
	in     io.Reader
	reader *bufio.Reader
	out    io.Writer
	buf    *bufio.Writer
	err    io.Writer
}

func NewStdio(in io.Reader, out, err io.Writer) *Stdio {
	return &Stdio{in: in, out: out, err: err}
}

func (s *Stdio) Out(b []byte) {
	if s.buf == nil {
		s.buf = bufio.NewWriterSize(s.out, 32<<10)
	}

	_, _ = s.buf.Write(b)
}

func (s *Stdio) Err(b []byte) {
	s.Flush()
	_, _ = s.err.Write(b)
}

func (s *Stdio) In() *bufio.Reader {
	s.Flush()

	if s.reader == nil {
		s.reader = bufio.NewReader(s.in)
	}

	return s.reader
}

func (s *Stdio) Flush() {
	if s.buf != nil {
		_ = s.buf.Flush()
	}
}

// trimEOL drops a trailing "\n" or "\r\n".
func trimEOL(line string) string {
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
}
