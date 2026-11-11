package errors

import (
	"fmt"
	"strings"
)

const maxTraceFrames = 10

// Report renders err for the terminal: the message, the source line it
// points at with a caret under the column, and for a runtime error the
// calls it unwound through.
func Report(err error, source string) string {
	var b strings.Builder

	b.WriteString(err.Error())
	b.WriteByte('\n')

	if p, ok := err.(interface{ Pos() (int, int) }); ok {
		line, col := p.Pos()
		writeSnippet(&b, source, line, col)
	}

	if rt, ok := err.(RuntimeError); ok && len(rt.frames) > 0 {
		writeTrace(&b, rt.token.Line, rt.frames)
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

type entry struct {
	name   string
	line   int
	native bool
	repeat int
}

func writeEntries(b *strings.Builder, entries []entry) {
	for _, e := range entries {
		if e.native {
			fmt.Fprintf(b, "  at %s (native)\n", e.name)
		} else {
			fmt.Fprintf(b, "  at %s (line %d)\n", e.name, e.line)
		}

		if e.repeat > 0 {
			fmt.Fprintf(b, "  ... repeated %d more times\n", e.repeat)
		}
	}
}

// writeTrace prints the frames innermost first; each frame's line is the
// line execution was at in it, which for an outer frame is where it made
// the call below it.
func writeTrace(b *strings.Builder, errLine int, frames []Frame) {
	entries := make([]entry, 0, len(frames)+1)
	line := errLine

	add := func(e entry) {
		if n := len(entries); n > 0 && entries[n-1].name == e.name && entries[n-1].line == e.line && entries[n-1].native == e.native {
			entries[n-1].repeat++
			return
		}

		entries = append(entries, e)
	}

	for _, f := range frames {
		add(entry{name: f.Name, line: line, native: f.Native})
		line = f.Line
	}

	add(entry{name: "<script>", line: line})

	if len(entries) > maxTraceFrames+1 {
		skipped := len(entries) - maxTraceFrames - 1
		writeEntries(b, entries[:maxTraceFrames])
		fmt.Fprintf(b, "  ... %d more\n", skipped)
		entries = entries[len(entries)-1:]
	}

	writeEntries(b, entries)
}
