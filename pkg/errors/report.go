package errors

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"
)

const maxTraceFrames = 10

// File is a source file of a program; Path is relative to the entry
// file's directory.
type File struct {
	Path   string
	Source string
}

// Files are a program's files, by a token's File; the entry file is first.
type Files []File

// where names a line of file.
func (f Files) where(file, line int) string {
	if file >= 0 && file < len(f) {
		return fmt.Sprintf("%s:%d", f[file].Path, line)
	}

	return fmt.Sprintf("line %d", line)
}

// lines are the lines of file, if the program has it.
func (f Files) lines(file int) []string {
	if file < 0 || file >= len(f) {
		return nil
	}

	return strings.Split(f[file].Source, "\n")
}

// style paints the parts of a report with ANSI colors, or leaves them plain.
type style bool

const (
	bold   = "1"
	red    = "1;31"
	blue   = "1;34"
	cyan   = "1;36"
	dimmed = "2"
)

func (s style) paint(code, text string) string {
	if !s {
		return text
	}

	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

// diagnostic is an error that knows what it is, where it points and how
// to fix it, if it is sure.
type diagnostic interface {
	error
	// heading is the kind of error, empty for a plain one, and message.
	heading() (string, string)
	span() Span
	hint() string
}

// ReportAll renders errs as Report does, a blank line apart.
func ReportAll(errs []error, files Files, color bool) string {
	reports := make([]string, len(errs))
	for i, err := range errs {
		reports[i] = Report(err, files, color)
	}

	return strings.Join(reports, "\n")
}

// Report renders err for the terminal: what went wrong, the source it
// points at, a fix when the error is sure of one, and for a runtime error
// the calls it unwound through. color paints it with ANSI colors.
func Report(err error, files Files, color bool) string {
	var b strings.Builder

	s := style(color)

	d, ok := err.(diagnostic)
	if !ok {
		fmt.Fprintf(&b, "%s%s\n", s.paint(red, "error"), s.paint(bold, ": "+err.Error()))
		return b.String()
	}

	kind, msg := d.heading()
	fmt.Fprintf(&b, "%s%s\n", s.paint(red, cmp.Or(kind, "error")), s.paint(bold, ": "+msg))
	writeSource(&b, s, files, d.span(), d.hint())

	if rt, ok := err.(RuntimeError); ok && len(rt.frames) > 0 {
		b.WriteByte('\n')
		writeTrace(&b, s, files, rt.token.File, rt.token.Line, rt.frames)
	}

	return b.String()
}

// writeSource shows where span is: its file, its line and the one before
// with the span underlined, then the hint.
func writeSource(b *strings.Builder, s style, files Files, span Span, hint string) {
	width := len(strconv.Itoa(span.Line))
	pad := strings.Repeat(" ", width)
	bar := s.paint(blue, pad+" |")
	lines := files.lines(span.File)

	fmt.Fprintf(b, "%s %s:%d\n", s.paint(blue, pad+"-->"), files.where(span.File, span.Line), span.Col)

	if text := line(lines, span.Line); strings.TrimSpace(text) != "" {
		fmt.Fprintln(b, bar)

		if before := line(lines, span.Line-1); strings.TrimSpace(before) != "" {
			fmt.Fprintf(b, "%s %s\n", s.paint(blue, fmt.Sprintf("%*d |", width, span.Line-1)), before)
		}

		fmt.Fprintf(b, "%s %s\n", s.paint(blue, fmt.Sprintf("%*d |", width, span.Line)), text)

		if span.Col > 0 {
			fmt.Fprintf(b, "%s %s%s\n", bar, indent(text, span.Col), s.paint(red, strings.Repeat("^", span.Width)))
		}
	}

	if hint != "" {
		fmt.Fprintln(b, bar)
		fmt.Fprintf(b, "%s %s %s\n", s.paint(blue, pad+" ="), s.paint(cyan, "hint:"), hint)
	}
}

// line is line n of lines, empty past them.
func line(lines []string, n int) string {
	if n < 1 || n > len(lines) {
		return ""
	}

	return strings.TrimRight(lines[n-1], "\r")
}

// indent is the blank before column col of text; it copies the tabs so the
// caret lines up under them.
func indent(text string, col int) string {
	var pad strings.Builder
	for i, r := range []rune(text) {
		if i >= col-1 {
			break
		}

		if r == '\t' {
			pad.WriteRune('\t')
		} else {
			pad.WriteByte(' ')
		}
	}

	if n := col - 1 - len([]rune(text)); n > 0 {
		pad.WriteString(strings.Repeat(" ", n))
	}

	return pad.String()
}

// site is a call in a trace: the function and where it was.
type site struct {
	name  string
	where string
}

type entry struct {
	site
	repeat int
}

func writeEntries(b *strings.Builder, s style, entries []entry) {
	for _, e := range entries {
		fmt.Fprintf(b, "  %s %s %s\n", s.paint(dimmed, "at"), e.name, s.paint(dimmed, "("+e.where+")"))

		if e.repeat > 0 {
			fmt.Fprintf(b, "  %s\n", s.paint(dimmed, fmt.Sprintf("... repeated %d more times", e.repeat)))
		}
	}
}

// writeTrace prints the frames innermost first; each frame's line is the
// line execution was at in it, which for an outer frame is where it made
// the call below it.
func writeTrace(b *strings.Builder, s style, files Files, errFile, errLine int, frames []Frame) {
	entries := make([]entry, 0, len(frames)+1)
	file, line := errFile, errLine

	add := func(st site) {
		if n := len(entries); n > 0 && entries[n-1].site == st {
			entries[n-1].repeat++
			return
		}

		entries = append(entries, entry{site: st})
	}

	for _, f := range frames {
		where := files.where(file, line)
		if f.Native {
			where = "native"
		}

		add(site{f.Name, where})
		file, line = f.File, f.Line
	}

	add(site{"<script>", files.where(file, line)})

	if len(entries) > maxTraceFrames+1 {
		skipped := len(entries) - maxTraceFrames - 1
		writeEntries(b, s, entries[:maxTraceFrames])
		fmt.Fprintf(b, "  %s\n", s.paint(dimmed, fmt.Sprintf("... %d more", skipped)))
		entries = entries[len(entries)-1:]
	}

	writeEntries(b, s, entries)
}
