package rune

import (
	stderrors "errors"
	"fmt"
	"io/fs"
	"path"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"slices"
	"strings"
)

// Load scans, parses and resolves the program whose entry file is name in
// fsys, along with the files it imports, which are named from the root of
// fsys, or relative to the importing file. Each file has its own top level,
// sees another's exports through the name it imports it as, and runs once,
// after the files it imports; then main, if the entry file declares it.
// root is where fsys is on disk, for module.dir. The files come back even
// on error, for reporting it.
func Load(fsys fs.FS, root, name string) (*Program, []errors.File, []error) {
	l := loader{fsys: fsys, index: map[string]int{}}
	l.visit(name, nil)

	if len(l.errors) > 0 {
		return nil, l.files, l.errors
	}

	r := NewResolver()
	r.root = root
	if errs := r.ResolveModules(l.modules); len(errs) > 0 {
		return nil, l.files, errs
	}

	var stmts []ast.Stmt
	for _, m := range l.modules {
		stmts = append(stmts, m.stmts...)
	}

	prog, err := compile(stmts, r.builtinLocals(), r.main, r.methodTable())
	if err != nil {
		return nil, l.files, []error{err}
	}

	return prog, l.files, nil
}

// loader walks the imports depth first, so modules come after the ones
// they import; stack holds the files still loading, the importers of the
// one being loaded.
type loader struct {
	fsys    fs.FS
	files   []errors.File
	index   map[string]int
	stack   []int
	modules []*fileAST
	errors  []error
}

// visit loads the file at path and, before it, the files it imports. The
// scanner skips bad characters, so parse errors are reported alongside its
// own, unless a string ran to the end of the file and left nothing worth
// parsing.
func (l *loader) visit(path string, from *importDecl) {
	if i, ok := l.index[path]; ok {
		if j := slices.Index(l.stack, i); j >= 0 {
			l.cycle(l.stack[j:], from)
		}

		return
	}

	src, err := fs.ReadFile(l.fsys, path)
	if err != nil {
		l.errors = append(l.errors, readError(err, from))
		return
	}

	i := len(l.files)
	l.files = append(l.files, errors.File{Path: path, Source: string(src)})
	l.index[path] = i

	scanner := scan(src, i)
	if scanner.truncated {
		l.errors = append(l.errors, scanner.errors...)
		return
	}

	file, parseErrors := parseFile(scanner.tokens, from != nil)
	l.errors = append(append(l.errors, scanner.errors...), parseErrors...)

	if file == nil {
		return
	}

	file.file, file.path = i, path

	l.stack = append(l.stack, i)
	for j := range file.imports {
		imp := &file.imports[j]

		p, ok := modulePath(path, imp)
		if !ok {
			l.errors = append(l.errors, errors.NewSyntaxError(imp.tok, fmt.Sprintf("Module '%s' is outside the program's directory.", imp.name)))
			continue
		}

		l.visit(p, imp)
		imp.file = l.index[p]
	}
	l.stack = l.stack[:len(l.stack)-1]

	l.modules = append(l.modules, file)
}

// modulePath is the file imp names, imported from the file at from; ok is
// false if it is outside the root.
func modulePath(from string, imp *importDecl) (p string, ok bool) {
	file := strings.ReplaceAll(imp.name[imp.dots:], ".", "/") + ".rn"
	if imp.dots == 0 {
		return file, true
	}

	p = path.Join(path.Dir(from), strings.Repeat("../", imp.dots-1), file)

	return p, p != ".." && !strings.HasPrefix(p, "../")
}

func readError(err error, from *importDecl) error {
	if from == nil {
		return err
	}

	if stderrors.Is(err, fs.ErrNotExist) {
		return errors.NewSyntaxError(from.tok, fmt.Sprintf("Cannot find module '%s'.", from.name))
	}

	return errors.NewSyntaxError(from.tok, fmt.Sprintf("Cannot read module '%s': %v.", from.name, err))
}

// cycle reports that from imports loop[0], which imports the rest of
// loop in turn, back to from.
func (l *loader) cycle(loop []int, from *importDecl) {
	chain := make([]string, 0, len(loop)+1)
	for _, f := range loop {
		chain = append(chain, l.files[f].Path)
	}

	chain = append(chain, chain[0])

	l.errors = append(l.errors, errors.NewSyntaxError(from.tok, "Import cycle: "+strings.Join(chain, " -> ")+"."))
}
