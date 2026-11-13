package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"rune/pkg/format"
	"strings"
)

func formatFile(stdout, stderr io.Writer, path string, src []byte, _ []string) int {
	out, errs := format.Source(src)
	if errs != nil {
		printErrors(stderr, errs, soleFile(path, src))
		return exitCodeParseError
	}

	stdout.Write(out)

	return exitCodeOk
}

// formatPaths formats the .rn files at paths, directories walked, and
// rewrites them, or with check lists the ones not formatted.
func formatPaths(stdout, stderr io.Writer, check bool, paths []string) int {
	if len(paths) == 0 {
		fmt.Fprintln(stderr, "Usage: rune fmt [-w | --check] <path>...")
		return exitCodeError
	}

	code := exitCodeOk

	for _, root := range paths {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, runeExtension) {
				return err
			}

			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			out, errs := format.Source(src)

			switch {
			case errs != nil:
				fmt.Fprintf(stderr, "%s:\n", path)
				printErrors(stderr, errs, soleFile(path, src))
				code = exitCodeParseError
			case bytes.Equal(src, out):
			case check:
				fmt.Fprintln(stdout, path)
				code = max(code, exitCodeError)
			default:
				return os.WriteFile(path, out, 0o644)
			}

			return nil
		})
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return exitCodeError
		}
	}

	return code
}
