package stdlib

import (
	"bufio"
	stderrors "errors"
	"io"
	"os"
	"rune/pkg/errors"
	"rune/pkg/value"
	"unsafe"
)

// Stream is an open file, or the body of a response, read or written in
// pieces. The host closes what the program leaves open when it ends.
type Stream struct {
	name   string
	r      *bufio.Reader
	w      *bufio.Writer
	closer io.Closer
	// file is the open file, nil for a body.
	file   *os.File
	closed bool
}

func (s *Stream) String() string { return "<stream " + s.name + ">" }

var errClosed = stderrors.New("stream is closed")

// ReadLine returns the next line without "\n" or "\r\n"; ok is false at
// the end.
func (s *Stream) ReadLine() (line string, ok bool, err error) {
	if err := s.readable(); err != nil {
		return "", false, err
	}

	line, err = s.r.ReadString('\n')
	if err == io.EOF {
		return trimEOL(line), line != "", nil
	}

	return trimEOL(line), err == nil, err
}

func (s *Stream) readable() error {
	switch {
	case s.closed:
		return errClosed
	case s.r == nil:
		return stderrors.New("stream is not open for reading")
	}

	return nil
}

// seekable checks that s is an open file, which readAt, seek and sync need.
func (s *Stream) seekable() error {
	switch {
	case s.closed:
		return errClosed
	case s.file == nil:
		return stderrors.New("stream is not a file")
	}

	return nil
}

func (s *Stream) writable() error {
	switch {
	case s.closed:
		return errClosed
	case s.w == nil:
		return stderrors.New("stream is not open for writing")
	}

	return nil
}

// flush writes out what s buffered, if it writes.
func (s *Stream) flush() error {
	if s.w == nil {
		return nil
	}

	return s.w.Flush()
}

// Close flushes and closes s; closing it again does nothing.
func (s *Stream) Close() error {
	if s.closed {
		return nil
	}

	s.closed = true

	return stderrors.Join(s.flush(), s.closer.Close())
}

// open registers s with the host, which closes it at the end if the
// program does not.
func (f *Host) open(s *Stream) value.Value {
	if f.streams == nil {
		f.streams = map[*Stream]struct{}{}
	}

	f.streams[s] = struct{}{}

	return value.NewStream(s)
}

// CloseStreams closes the streams the program left open.
func (f *Host) CloseStreams() {
	for s := range f.streams {
		_ = s.Close()
	}

	clear(f.streams)
}

var openModes = map[string]int{
	"r": os.O_RDONLY,
	"w": os.O_WRONLY | os.O_CREATE | os.O_TRUNC,
	"a": os.O_WRONLY | os.O_CREATE | os.O_APPEND,
}

func openFile(c *call) value.Value {
	path, mode := c.str(0), "r"
	if c.has(1) {
		mode = c.str(1)
	}

	if !c.ok() {
		return value.NilValue
	}

	flag, ok := openModes[mode]
	if !ok {
		return c.fail(errors.ValueError, "expects mode \"r\", \"w\" or \"a\", got %q.", mode)
	}

	f, err := os.OpenFile(path, flag, 0o644)
	if err != nil {
		return c.ioFail(err)
	}

	return c.host.open(fileStream(path, f, mode == "r"))
}

// fileStream is a stream of f, for reading or for writing.
func fileStream(path string, f *os.File, read bool) *Stream {
	s := &Stream{name: path, closer: f, file: f}
	if read {
		s.r = bufio.NewReader(f)
	} else {
		s.w = bufio.NewWriter(f)
	}

	return s
}

func (c *call) stream(i int) *Stream {
	if c.args[i].Kind() != value.Stream {
		c.typeError(i, "a stream")
		return nil
	}

	return c.args[i].Internal().(*Stream)
}

// streamFail reports err of stream s as an IOError naming it.
func (c *call) streamFail(s *Stream, err error) value.Value {
	if c.err == nil {
		c.err = errors.NewIOError(*c.token, c.n.name+"() "+s.name+": "+err.Error()+".", s.name, ioCode(err))
	}

	return value.NilValue
}

// streamMethod runs fn on the stream receiver once ready, which checks
// that it is open for reading or writing, says it can go ahead.
func streamMethod(name string, min, max int, ready func(*Stream) error, fn func(c *call, s *Stream) value.Value) *Native {
	return method(name, min, max, (*call).stream, func(c *call, s *Stream) value.Value {
		if err := ready(s); err != nil {
			return c.streamFail(s, err)
		}

		return fn(c, s)
	})
}

func reading(name string, min, max int, fn func(c *call, s *Stream) value.Value) *Native {
	return streamMethod(name, min, max, (*Stream).readable, fn)
}

// readRest reads the rest of the stream, which make wraps.
func readRest(name string, make func([]byte) value.Value) *Native {
	return reading(name, 1, 1, func(c *call, s *Stream) value.Value {
		b, err := io.ReadAll(s.r)
		if err != nil {
			return c.streamFail(s, err)
		}

		return make(b)
	})
}

// maxRead caps the size of one Stream.read.
const maxRead = 1 << 30

var streamMethods = []*Native{
	reading("stream.readLine", 1, 1, func(c *call, s *Stream) value.Value {
		line, ok, err := s.ReadLine()
		if err != nil {
			return c.streamFail(s, err)
		}

		if !ok {
			return value.NilValue
		}

		return value.NewString(line)
	}),
	reading("stream.read", 2, 2, func(c *call, s *Stream) value.Value {
		n := c.int(1)
		if !c.ok() {
			return value.NilValue
		}

		if n <= 0 || n > maxRead {
			return c.fail(errors.ValueError, "expects a size from 1 to %d, got %d.", maxRead, n)
		}

		// A small read takes what is buffered, not the whole n.
		buf := make([]byte, min(n, max(s.r.Buffered(), 4096)))

		got, err := s.r.Read(buf)
		if got == 0 && err == io.EOF {
			return value.NilValue
		}

		if err != nil && err != io.EOF {
			return c.streamFail(s, err)
		}

		return value.NewBytes(buf[:got:got])
	}),
	readRest("stream.readBytes", value.NewBytes),
	readRest("stream.readText", func(b []byte) value.Value { return value.NewString(bytesString(b)) }),
	streamMethod("stream.write", 2, 2, (*Stream).writable, func(c *call, s *Stream) value.Value {
		data := c.data(1)
		if !c.ok() {
			return value.NilValue
		}

		if _, err := s.w.Write(data); err != nil {
			return c.streamFail(s, err)
		}

		return value.NilValue
	}),
	streamMethod("stream.flush", 1, 1, (*Stream).writable, func(c *call, s *Stream) value.Value {
		if err := s.flush(); err != nil {
			return c.streamFail(s, err)
		}

		return value.NilValue
	}),
	// readAt reads up to n bytes at offset, past what is buffered; nil at
	// or past the end.
	streamMethod("stream.readAt", 3, 3, (*Stream).seekable, func(c *call, s *Stream) value.Value {
		at, n := c.int(1), c.int(2)
		switch {
		case !c.ok():
			return value.NilValue
		case at < 0:
			return c.fail(errors.ValueError, "expects an offset of at least 0, got %d.", at)
		case n <= 0 || n > maxRead:
			return c.fail(errors.ValueError, "expects a size from 1 to %d, got %d.", maxRead, n)
		}

		// A large read is cut to the file, so it does not allocate n.
		if n > 1<<16 {
			info, err := s.file.Stat()
			if err != nil {
				return c.streamFail(s, err)
			}

			n = int(max(0, min(int64(n), info.Size()-int64(at))))
		}

		buf := make([]byte, n)

		got, err := s.file.ReadAt(buf, int64(at))
		if err != nil && err != io.EOF {
			return c.streamFail(s, err)
		}

		if got == 0 {
			return value.NilValue
		}

		return value.NewBytes(buf[:got:got])
	}),
	// seek moves to offset from the start, flushing what is written first.
	streamMethod("stream.seek", 2, 2, (*Stream).seekable, func(c *call, s *Stream) value.Value {
		at := c.int(1)
		switch {
		case !c.ok():
			return value.NilValue
		case at < 0:
			return c.fail(errors.ValueError, "expects an offset of at least 0, got %d.", at)
		}

		if err := s.flush(); err != nil {
			return c.streamFail(s, err)
		}

		if _, err := s.file.Seek(int64(at), io.SeekStart); err != nil {
			return c.streamFail(s, err)
		}

		if s.r != nil {
			s.r.Reset(s.file)
		}

		return value.NilValue
	}),
	// sync flushes and has the system write the file to disk.
	streamMethod("stream.sync", 1, 1, (*Stream).seekable, func(c *call, s *Stream) value.Value {
		if err := stderrors.Join(s.flush(), s.file.Sync()); err != nil {
			return c.streamFail(s, err)
		}

		return value.NilValue
	}),
	method("stream.close", 1, 1, (*call).stream, func(c *call, s *Stream) value.Value {
		delete(c.host.streams, s)

		if err := s.Close(); err != nil {
			return c.streamFail(s, err)
		}

		return value.NilValue
	}),
}

// data is argument i as the bytes to write: a string's or bytes'.
func (c *call) data(i int) []byte {
	v := c.args[i]
	if v.IsString() {
		// Writers only read it, so the string need not be copied.
		return unsafe.Slice(unsafe.StringData(v.Str()), len(v.Str()))
	}

	if b, ok := v.Binary(); ok {
		return b
	}

	c.typeError(i, "a string, bytes or a buffer")

	return nil
}
