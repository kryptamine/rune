package stdlib

import (
	stderrors "errors"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"rune/pkg/errors"
	"rune/pkg/value"
	"strings"
	"syscall"
	"unsafe"
)

var FileInfo = value.NewShape("FileInfo", []value.Field{
	{Name: "name", Required: true},
	{Name: "size", Required: true},
	{Name: "isDir", Required: true},
	{Name: "modified", Required: true},
	{Name: "mode", Required: true},
})

// Types are the built-in struct types, which programs see by name.
var Types = append(errors.Kinds[:len(errors.Kinds):len(errors.Kinds)], FileInfo)

var fsNatives = []*Native{
	{"read", 1, 1, func(c *call) value.Value {
		b, err := os.ReadFile(c.str(0))
		if c.ok() && err != nil {
			return c.ioFail(err)
		}

		return value.NewString(bytesString(b))
	}},
	{"lines", 1, 1, func(c *call) value.Value {
		b, err := os.ReadFile(c.str(0))
		if c.ok() && err != nil {
			return c.ioFail(err)
		}

		s := strings.TrimSuffix(bytesString(b), "\n")
		if s == "" {
			return value.NewArray(nil)
		}

		items := make([]value.Value, 0, strings.Count(s, "\n")+1)
		for line := range strings.SplitSeq(s, "\n") {
			items = append(items, value.NewString(strings.TrimSuffix(line, "\r")))
		}

		return value.NewArray(items)
	}},
	{"write", 2, 2, func(c *call) value.Value {
		path, s := c.str(0), c.str(1)
		if c.ok() {
			c.check(os.WriteFile(path, unsafe.Slice(unsafe.StringData(s), len(s)), 0o644))
		}

		return value.NilValue
	}},
	{"append", 2, 2, func(c *call) value.Value {
		path, s := c.str(0), c.str(1)
		if !c.ok() {
			return value.NilValue
		}

		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return c.ioFail(err)
		}

		_, err = f.WriteString(s)
		c.check(stderrors.Join(err, f.Close()))

		return value.NilValue
	}},
	{"exists", 1, 1, func(c *call) value.Value {
		path := c.str(0)
		if !c.ok() {
			return value.NilValue
		}

		_, err := os.Stat(path)
		if err != nil && !stderrors.Is(err, iofs.ErrNotExist) {
			return c.ioFail(err)
		}

		return value.NewBool(err == nil)
	}},
	{"stat", 1, 1, func(c *call) value.Value {
		path := c.str(0)
		if !c.ok() {
			return value.NilValue
		}

		info, err := os.Stat(path)
		if err != nil {
			return c.ioFail(err)
		}

		return fileInfo(info)
	}},
	// list returns the entries of a directory, sorted by name.
	{"list", 1, 1, func(c *call) value.Value {
		path := c.str(0)
		if !c.ok() {
			return value.NilValue
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return c.ioFail(err)
		}

		items := make([]value.Value, len(entries))

		for i, e := range entries {
			info, err := e.Info()
			if err != nil {
				return c.ioFail(err)
			}

			items[i] = fileInfo(info)
		}

		return value.NewArray(items)
	}},
	{"mkdir", 1, 1, func(c *call) value.Value {
		if path := c.str(0); c.ok() {
			c.check(os.MkdirAll(path, 0o755))
		}

		return value.NilValue
	}},
	{"remove", 1, 1, func(c *call) value.Value {
		if path := c.str(0); c.ok() {
			c.check(os.Remove(path))
		}

		return value.NilValue
	}},
	{"removeAll", 1, 1, func(c *call) value.Value {
		path := c.str(0)
		if !c.ok() {
			return value.NilValue
		}

		if path == "" {
			return c.fail(errors.ValueError, "got an empty path.")
		}

		c.check(os.RemoveAll(path))

		return value.NilValue
	}},
	{"rename", 2, 2, func(c *call) value.Value {
		if from, to := c.str(0), c.str(1); c.ok() {
			c.check(os.Rename(from, to))
		}

		return value.NilValue
	}},
	{"copy", 2, 2, func(c *call) value.Value {
		if from, to := c.str(0), c.str(1); c.ok() {
			c.check(copyFile(from, to))
		}

		return value.NilValue
	}},
	{"cwd", 0, 0, func(c *call) value.Value {
		dir, err := os.Getwd()
		if err != nil {
			return c.ioFail(err)
		}

		return value.NewString(dir)
	}},
	// tempDir creates a new directory, which the program removes.
	{"tempDir", 0, 0, func(c *call) value.Value {
		dir, err := os.MkdirTemp("", "rune-")
		if err != nil {
			return c.ioFail(err)
		}

		return value.NewString(dir)
	}},
	{"join", 1, -1, func(c *call) value.Value {
		parts := make([]string, len(c.args))
		for i := range c.args {
			parts[i] = c.str(i)
		}

		return value.NewString(filepath.Join(parts...))
	}},
	pathNative("base", filepath.Base),
	pathNative("dir", filepath.Dir),
	pathNative("ext", filepath.Ext),
}

func pathNative(name string, f func(string) string) *Native {
	return &Native{name, 1, 1, func(c *call) value.Value {
		return value.NewString(f(c.str(0)))
	}}
}

func fileInfo(info iofs.FileInfo) value.Value {
	v, fields := value.NewStruct(FileInfo)
	fields[0] = value.NewString(info.Name())
	fields[1] = count(int(info.Size()))
	fields[2] = value.NewBool(info.IsDir())
	fields[3] = value.NewNumber(float64(info.ModTime().UnixNano()) / 1e9)
	fields[4] = count(int(info.Mode().Perm()))

	return v
}

func copyFile(from, to string) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}

	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}

	if info.IsDir() {
		return &iofs.PathError{Op: "copy", Path: from, Err: syscall.EISDIR}
	}

	dst, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}

	_, err = io.Copy(dst, src)

	return stderrors.Join(err, dst.Close())
}

// bytesString takes ownership of b, which nothing else may hold.
func bytesString(b []byte) string { return unsafe.String(unsafe.SliceData(b), len(b)) }

func (c *call) check(err error) {
	if err != nil {
		c.ioFail(err)
	}
}

func (c *call) ioFail(err error) value.Value {
	if c.err != nil {
		return value.NilValue
	}

	path, reason := "", err.Error()

	var pe *iofs.PathError

	var le *os.LinkError

	switch {
	case stderrors.As(err, &pe):
		path, reason = pe.Path, pe.Path+": "+pe.Err.Error()
	case stderrors.As(err, &le):
		path, reason = le.Old, le.Old+" -> "+le.New+": "+le.Err.Error()
	}

	c.err = errors.NewIOError(*c.token, c.name+"() "+reason+".", path, ioCode(err))

	return value.NilValue
}

func ioCode(err error) string {
	// ENOTEMPTY is also ErrExist, so the errnos go first.
	switch {
	case stderrors.Is(err, syscall.ENOTEMPTY):
		return "notEmpty"
	case stderrors.Is(err, syscall.EISDIR):
		return "isDir"
	case stderrors.Is(err, syscall.ENOTDIR):
		return "notDir"
	case stderrors.Is(err, iofs.ErrNotExist):
		return "notFound"
	case stderrors.Is(err, iofs.ErrExist):
		return "exists"
	case stderrors.Is(err, iofs.ErrPermission):
		return "permission"
	}

	return ""
}
