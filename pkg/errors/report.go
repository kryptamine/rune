package errors

import (
	"fmt"
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

// path names file; the entry file goes unnamed.
func (f Files) path(file int) (string, bool) {
	if file > 0 && file < len(f) {
		return f[file].Path, true
	}

	return "", false
}

// where names a line of file.
func (f Files) where(file, line int) string {
	if path, ok := f.path(file); ok {
		return fmt.Sprintf("%s:%d", path, line)
	}

	return fmt.Sprintf("line %d", line)
}

// Report renders err for the terminal: the message, the source line it
// points at with a caret under the column, and for a runtime error the
// calls it unwound through.
func Report(err error, files Files) string {
	var b strings.Builder

	b.WriteString(err.Error())
	b.WriteByte('\n')

	if p, ok := err.(Positioned); ok {
		file, line, col := p.Pos()
		if path, ok := files.path(file); ok {
			fmt.Fprintf(&b, "  --> %s:%d:%d\n", path, line, col)
		}

		if file < len(files) {
			writeSnippet(&b, files[file].Source, line, col)
		}
	}

	if r, ok := err.(RedeclaredError); ok {
		fmt.Fprintf(&b, "  note: first declared at %s\n", files.where(r.first.File, r.first.Line))
	}

	if rt, ok := err.(RuntimeError); ok && len(rt.frames) > 0 {
		writeTrace(&b, files, rt.token.File, rt.token.Line, rt.frames)
	}

	return b.String()
}

func writeSnippet(b *strings.Builder, source string, line, col int) {
	lines := strings.Split(source, "\n")
	if line < 1 || line > len(lines) {
		return
	}

	text := strings.TrimRight(lines[line-1], "\r")
	if strings.TrimSpace(text) == "" {
		return
	}

	gutter := fmt.Sprintf("%5d | ", line)
	fmt.Fprintf(b, "%s%s\n", gutter, text)

	if col < 1 {
		return
	}

	// Copy tabs from the source so the caret lines up under them.
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

	fmt.Fprintf(b, "%s| %s^\n", strings.Repeat(" ", len(gutter)-2), pad.String())
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

func writeEntries(b *strings.Builder, entries []entry) {
	for _, e := range entries {
		fmt.Fprintf(b, "  at %s (%s)\n", e.name, e.where)

		if e.repeat > 0 {
			fmt.Fprintf(b, "  ... repeated %d more times\n", e.repeat)
		}
	}
}

// writeTrace prints the frames innermost first; each frame's line is the
// line execution was at in it, which for an outer frame is where it made
// the call below it.
func writeTrace(b *strings.Builder, files Files, errFile, errLine int, frames []Frame) {
	entries := make([]entry, 0, len(frames)+1)
	file, line := errFile, errLine

	add := func(s site) {
		if n := len(entries); n > 0 && entries[n-1].site == s {
			entries[n-1].repeat++
			return
		}

		entries = append(entries, entry{site: s})
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
		writeEntries(b, entries[:maxTraceFrames])
		fmt.Fprintf(b, "  ... %d more\n", skipped)
		entries = entries[len(entries)-1:]
	}

	writeEntries(b, entries)
}
