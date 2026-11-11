package rune

// Load scans, parses and resolves a program. The scanner skips bad
// characters, so parse errors are reported alongside its own, unless a
// string ran to the end of the file and left nothing worth parsing.
func Load(source []byte) (*Program, []error) {
	scanner := scan(source)
	if scanner.truncated {
		return nil, scanner.errors
	}

	stmts, parseErrors := ParseStmts(scanner.tokens)
	if errs := append(scanner.errors, parseErrors...); len(errs) > 0 {
		return nil, errs
	}

	if errs := NewResolver().ResolveStmts(stmts); len(errs) > 0 {
		return nil, errs
	}

	return Compile(stmts), nil
}
