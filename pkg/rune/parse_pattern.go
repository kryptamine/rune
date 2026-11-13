package rune

import (
	"rune/pkg/ast"
	"rune/pkg/errors"
)

// bindingOrPattern parses a binding, or a pattern binding the variable
// named hidden, which the statement returned destructures.
func (s *Parser) bindingOrPattern(hiddenName, msg string) (ast.Binding, ast.Stmt, error) {
	hidden := func() ast.Binding {
		name := s.previous()
		name.TokenType = ast.IDENTIFIER
		name.Lexeme = hiddenName

		return ast.Binding{Name: name}
	}

	switch {
	case s.match(ast.LEFT_BRACE):
		d, err := s.pattern(false)
		if err != nil {
			return ast.Binding{}, nil, err
		}

		b := hidden()
		d.Initializer = ast.NewVarExpr(b.Name)

		return b, d, nil
	case s.match(ast.LEFT_BRACKET):
		d, err := s.arrayPattern(false)
		if err != nil {
			return ast.Binding{}, nil, err
		}

		b := hidden()
		d.Initializer = ast.NewVarExpr(b.Name)

		return b, d, nil
	default:
		b, err := s.binding(msg)
		return b, nil, err
	}
}

// arrayPattern parses [a, mut b, _, ...rest]; the "[" has already been
// consumed. With mutable, every variable is mutable.
func (s *Parser) arrayPattern(mutable bool) (*ast.ArrayDestructureStmt, error) {
	d := &ast.ArrayDestructureStmt{Bracket: s.previous()}

	sawDefault := false

	for !s.check(ast.RIGHT_BRACKET) && !s.isAtEnd() {
		rest := s.match(ast.DOT_DOT_DOT)

		b, err := s.binding("Expect variable name.")
		if err != nil {
			return nil, err
		}

		b.Mutable = b.Mutable || mutable
		skip := b.Discards()

		def, err := s.patternDefault(rest || skip)
		if err != nil {
			return nil, err
		}

		if def == nil && sawDefault && !rest {
			s.errors = append(s.errors, errors.NewSyntaxError(b.Name, "An element without a default cannot follow one with a default."))
		}

		sawDefault = sawDefault || def != nil

		v := &ast.PatternVar{Binding: b, Default: def}

		switch {
		case rest:
			d.Rest = v
		case skip:
			d.Elems = append(d.Elems, nil)
		default:
			d.Elems = append(d.Elems, v)
		}

		if rest && !s.check(ast.RIGHT_BRACKET) {
			return nil, errors.NewSyntaxError(s.peek(), "Rest element must be last.")
		}

		if !s.match(ast.COMMA) {
			break
		}
	}

	if len(d.Elems) == 0 && d.Rest == nil {
		s.errors = append(s.errors, errors.NewSyntaxError(s.peek(), "Expect variable name in pattern."))
	}

	_, err := s.consume(ast.RIGHT_BRACKET, "Expect ']' after pattern.")

	return d, err
}

// pattern parses { a, mut b, c: d, "e": f, ...rest }; the "{" has
// already been consumed. With mutable, every variable is mutable.
func (s *Parser) pattern(mutable bool) (*ast.DestructureStmt, error) {
	d := &ast.DestructureStmt{Brace: s.previous()}

	for !s.check(ast.RIGHT_BRACE) && !s.isAtEnd() {
		if s.match(ast.DOT_DOT_DOT) {
			b, err := s.binding("Expect variable name.")
			if err != nil {
				return nil, err
			}

			if _, err := s.patternDefault(true); err != nil {
				return nil, err
			}

			b.Mutable = b.Mutable || mutable
			d.Rest = &ast.PatternVar{Binding: b}

			if !s.check(ast.RIGHT_BRACE) {
				return nil, errors.NewSyntaxError(s.peek(), "Rest element must be last.")
			}

			break
		}

		f, err := s.patternField()
		if err != nil {
			return nil, err
		}

		f.Mutable = f.Mutable || mutable

		d.Fields = append(d.Fields, f)

		if !s.match(ast.COMMA) {
			break
		}
	}

	if len(d.Fields) == 0 && d.Rest == nil {
		s.errors = append(s.errors, errors.NewSyntaxError(s.peek(), "Expect property name in pattern."))
	}

	_, err := s.consume(ast.RIGHT_BRACE, "Expect '}' after pattern.")

	return d, err
}

// patternDefault parses the "= value" of the pattern element just parsed,
// if any; a rest element or _ cannot have one.
func (s *Parser) patternDefault(notAllowed bool) (ast.Expr, error) {
	if !s.match(ast.EQUAL) {
		return nil, nil
	}

	if notAllowed {
		return nil, errors.NewSyntaxError(s.previous(), "A rest element or _ cannot have a default.")
	}

	return s.expressionWithStructs(structsAll)
}

func (s *Parser) patternField() (ast.DestructureField, error) {
	f, err := s.patternKey()
	if err != nil {
		return f, err
	}

	f.Default, err = s.patternDefault(f.Discards())

	return f, err
}

func (s *Parser) patternKey() (ast.DestructureField, error) {
	if s.check(ast.MUT) {
		b, err := s.binding("Expect property name.")
		return ast.DestructureField{Key: b.Name, PatternVar: ast.PatternVar{Binding: b}}, err
	}

	if !s.match(ast.IDENTIFIER, ast.STRING) {
		return ast.DestructureField{}, errors.NewSyntaxError(s.peek(), "Expect property name.")
	}

	key := s.previous()
	quoted := key.TokenType == ast.STRING

	if quoted {
		key.Lexeme = key.Literal
	}

	if !s.match(ast.COLON) {
		if quoted {
			return ast.DestructureField{}, errors.NewSyntaxError(s.peek(), "Expect ':' after string key.")
		}

		return ast.DestructureField{Key: key, PatternVar: ast.PatternVar{Binding: ast.Binding{Name: key}}}, nil
	}

	b, err := s.binding("Expect variable name.")

	return ast.DestructureField{Key: key, PatternVar: ast.PatternVar{Binding: b}}, err
}

// binding parses "name" or "mut name".
func (s *Parser) binding(msg string) (ast.Binding, error) {
	mutable := s.match(ast.MUT)
	name, err := s.consume(ast.IDENTIFIER, msg)

	return ast.Binding{Name: name, Mutable: mutable}, err
}

// bindingEnd returns the index of the token after a binding starting i
// tokens ahead, or -1 if there is none.
func (s *Parser) bindingEnd(i int) int {
	if s.peekAt(i) == ast.MUT {
		i++
	}

	if s.peekAt(i) != ast.IDENTIFIER {
		return -1
	}

	return i + 1
}

// patternInitializer parses "= value;" after the pattern of a let.
func (s *Parser) patternInitializer() (ast.Expr, error) {
	if _, err := s.consume(ast.EQUAL, "Expect '=' after pattern."); err != nil {
		return nil, err
	}

	initializer, err := s.expression()
	if err != nil {
		return nil, err
	}

	_, err = s.consume(ast.SEMICOLON, "Expect ';' after variable declaration.")

	return initializer, err
}
