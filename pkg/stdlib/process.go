package stdlib

import (
	"bytes"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"runtime"
	"strings"
)

// Exit ends a program with Code, from os.exit or main returning it; try
// does not catch it.
type Exit struct{ Code int }

func (e Exit) Error() string { return fmt.Sprintf("exit status %d", e.Code) }

var ExecOptions = value.NewShape("ExecOptions", []value.Field{
	{Name: "input"},
	{Name: "dir"},
	{Name: "env"},
})

var ExecResult = value.NewShape("ExecResult", []value.Field{
	{Name: "code", Required: true},
	{Name: "stdout", Required: true},
	{Name: "stderr", Required: true},
})

func writer(name string, newline bool, write func(*Stdio, []byte)) *Native {
	return &Native{name, 1, 1, func(c *call) value.Value {
		b := helpers.AppendValue(nil, c.args[0])
		if newline {
			b = append(b, '\n')
		}

		write(c.host.Stdio, b)

		return value.NilValue
	}}
}

var ioNatives = []*Native{
	{"io.readLine", 0, 0, func(c *call) value.Value {
		line, err := c.host.Stdio.In().ReadString('\n')
		if err != nil && err != io.EOF {
			return c.fail(errors.IOError, "could not read stdin: %s.", err)
		}

		if line == "" {
			return value.NilValue
		}

		return value.NewString(trimEOL(line))
	}},
	{"io.read", 0, 0, func(c *call) value.Value {
		b, err := io.ReadAll(c.host.Stdio.In())
		if err != nil {
			return c.fail(errors.IOError, "could not read stdin: %s.", err)
		}

		return value.NewString(bytesString(b))
	}},
	writer("io.write", false, (*Stdio).Out),
	writer("io.ewrite", false, (*Stdio).Err),
	writer("io.eprint", true, (*Stdio).Err),
}

var osNatives = []*Native{
	{"os.env", 1, 1, func(c *call) value.Value {
		if v, ok := os.LookupEnv(c.str(0)); ok && c.ok() {
			return value.NewString(v)
		}

		return value.NilValue
	}},
	{"os.setEnv", 2, 2, func(c *call) value.Value {
		if name, v := c.str(0), c.str(1); c.ok() {
			c.envCheck(os.Setenv(name, v))
		}

		return value.NilValue
	}},
	{"os.unsetEnv", 1, 1, func(c *call) value.Value {
		if name := c.str(0); c.ok() {
			c.envCheck(os.Unsetenv(name))
		}

		return value.NilValue
	}},
	{"os.environ", 0, 0, func(*call) value.Value {
		env := os.Environ()
		obj := make(map[string]value.Value, len(env))

		for _, kv := range env {
			k, v, _ := strings.Cut(kv, "=")
			obj[k] = value.NewString(v)
		}

		return value.NewObject(obj)
	}},
	{"os.exit", 0, 1, func(c *call) value.Value {
		code := 0
		if c.has(0) {
			code = c.int(0)
		}

		if !c.ok() {
			return value.NilValue
		}

		if code < 0 || code > 255 {
			return c.fail(errors.ValueError, "expects an exit code from 0 to 255, got %d.", code)
		}

		c.err = Exit{code}

		return value.NilValue
	}},
	{"os.exec", 1, 3, func(c *call) value.Value {
		cmd := exec.CommandContext(c.host.ctx, c.str(0))

		if c.has(1) {
			for i, a := range c.arr(1) {
				if !a.IsString() {
					return c.fail(errors.TypeError, "expects an array of strings as argument 2, got %s at %d.", helpers.TypeName(a), i)
				}

				cmd.Args = append(cmd.Args, a.Str())
			}
		}

		if c.has(2) {
			c.execOptions(cmd)
		}

		if !c.ok() {
			return value.NilValue
		}

		var stdout, stderr bytes.Buffer

		cmd.Stdout, cmd.Stderr = &stdout, &stderr

		code := 0

		var ee *exec.ExitError

		switch err := cmd.Run(); {
		case err == nil:
		case stderrors.As(err, &ee):
			code = ee.ExitCode()
		case stderrors.Is(err, exec.ErrNotFound):
			c.err = errors.NewIOError(*c.token, c.name+"() "+err.Error()+".", cmd.Path, "notFound")
			return value.NilValue
		default:
			return c.ioFail(err)
		}

		v, fields := value.NewStruct(ExecResult)
		fields[0] = count(code)
		fields[1] = value.NewString(stdout.String())
		fields[2] = value.NewString(stderr.String())

		return v
	}},
}

var osConsts = []Builtin{
	{"platform", value.NewString(runtime.GOOS)},
}

func (c *call) envCheck(err error) {
	if err != nil {
		c.fail(errors.ValueError, "%s.", err)
	}
}

// execOptions applies argument 3, an ExecOptions or an object of its fields, to cmd.
func (c *call) execOptions(cmd *exec.Cmd) {
	opts, err := Convert(c.token, ExecOptions, c.args[2])
	if err != nil {
		c.err = err
		return
	}

	f := opts.Fields()

	if input, ok := c.option(ExecOptions, f, 0, value.String, "a string"); ok {
		cmd.Stdin = strings.NewReader(input.Str())
	}

	if dir, ok := c.option(ExecOptions, f, 1, value.String, "a string"); ok {
		cmd.Dir = dir.Str()
	}

	if env, ok := c.option(ExecOptions, f, 2, value.Object, "an object"); ok {
		cmd.Env = os.Environ()

		for name, v := range env.Obj() {
			if !v.IsString() {
				c.fail(errors.TypeError, "expects env '%s' to be a string, got %s.", name, helpers.TypeName(v))
				return
			}

			cmd.Env = append(cmd.Env, name+"="+v.Str())
		}
	}
}

// option returns field i of an options struct of shape if it is set; it must be of kind.
func (c *call) option(shape *value.Shape, fields []value.Value, i int, kind value.Kind, want string) (value.Value, bool) {
	v := fields[i]
	if v.IsNil() || !c.ok() {
		return v, false
	}

	if v.Kind() != kind {
		c.fail(errors.TypeError, "expects %s.%s to be %s, got %s.", shape.Name, shape.Fields[i], want, helpers.TypeName(v))
		return v, false
	}

	return v, true
}
