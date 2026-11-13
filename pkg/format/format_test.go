package format

import (
	"io/fs"
	"os"
	"path/filepath"
	"rune/pkg/rune"
	"strings"
	"testing"
)

func TestRepoSources(t *testing.T) {
	for _, dir := range []string{"../../examples", "../../test"} {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".rn") {
				return err
			}
			src, _ := os.ReadFile(path)
			if _, errs := rune.Scan(src); len(errs) > 0 {
				return nil
			}

			once, errs := Source(src)
			if errs != nil {
				t.Errorf("%s: %v", path, errs)
				return nil
			}
			twice, errs := Source(once)
			if errs != nil || string(twice) != string(once) {
				t.Errorf("%s: not idempotent", path)
			}
			return nil
		})
	}
}
