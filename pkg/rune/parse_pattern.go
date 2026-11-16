package rune

import (
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

// bindingOrPattern parses a binding, or a pattern binding the variable
// named hidden, which the statement returned destructures.
func (s *Parser) bindingOrPattern(hiddenName, msg string) (ast.Binding, ast.Stmt, error) {
	if !s.check(ast.LEFT_BRACE) && !s.check(ast.LEFT_BRACKET) {
		b, err := s.binding(msg)
		return b, nil, err
	}

	name := s.peek()
	name.TokenType = ast.IDENTIFIER
	name.Lexeme = hiddenName

	pat, err := s.destructuring()
	if err != nil {
		return ast.Binding{}, nil, err
	}

	return ast.Binding{Name: name}, &ast.PatternStmt{Pattern: pat, Initializer: ast.NewVarExpr(name)}, nil
}

// destructuring parses the object or array pattern of a let, a parameter
// or a for-in variable, which must bind something.
func (s *Parser) destructuring() (ast.Pattern, error) {
	pat, err := s.patternAtom()
	if err != nil {
		return nil, err
	}

	if a, ok := pat.(*ast.ArrayPattern); ok && len(a.Elems) == 0 && !a.Open {
		s.errors = append(s.errors, errors.NewSyntaxError(s.previous(), "Expect variable name in pattern."))
	}

	return pat, nil
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

// matchPattern parses a pattern: alternatives separated by "|".
func (s *Parser) matchPattern() (ast.Pattern, error) {
	return s.alternatives(s.patternAlt)
}

func (s *Parser) alternatives(alt func() (ast.Pattern, error)) (ast.Pattern, error) {
	p, err := alt()
	if err != nil || !s.check(ast.PIPE) {
		return p, err
	}

	alts := []ast.Pattern{p}

	for s.match(ast.PIPE) {
		a, err := alt()
		if err != nil {
			return nil, err
		}

		alts = append(alts, a)
	}

	return &ast.OrPattern{Alts: alts}, nil
}

// patternAlt parses a pattern, or a range whose bounds are patterns: a
// number or the name of a constant, which the resolver checks.
func (s *Parser) patternAlt() (ast.Pattern, error) {
	var lo ast.Pattern

	if !s.check(ast.DOT_DOT) && !s.check(ast.DOT_DOT_EQUAL) {
		var err error
		if lo, err = s.patternAtom(); err != nil {
			return nil, err
		}
	}

	return s.rangeTail(lo)
}

// rangeTail parses the rest of a range pattern from lo, if one follows.
func (s *Parser) rangeTail(lo ast.Pattern) (ast.Pattern, error) {
	if !s.match(ast.DOT_DOT, ast.DOT_DOT_EQUAL) {
		return lo, nil
	}

	pat := &ast.RangePattern{Op: s.previous(), Lo: lo, Inclusive: s.previous().TokenType == ast.DOT_DOT_EQUAL}

	if s.check(ast.NUMBER) || s.check(ast.MINUS) || s.check(ast.IDENTIFIER) {
		var err error
		if pat.Hi, err = s.patternAtom(); err != nil {
			return nil, err
		}
	}

	switch {
	case pat.Hi == nil && pat.Inclusive:
		return nil, errors.NewSyntaxError(s.peek(), "Expect upper bound after '..='.")
	case pat.Hi == nil && lo == nil:
		return nil, errors.NewSyntaxError(s.peek(), "Expect a bound of the range pattern.")
	}

	return pat, nil
}

func (s *Parser) patternAtom() (ast.Pattern, error) {
	switch {
	case s.match(ast.NUMBER):
		return s.numberPattern(s.previous(), false)
	case s.match(ast.MINUS):
		minus := s.previous()
		if _, err := s.consume(ast.NUMBER, "Expect number after '-'."); err != nil {
			return nil, err
		}

		p, err := s.numberPattern(s.previous(), true)
		if p != nil {
			p.(*ast.LiteralPattern).Token = minus
		}

		return p, err
	case s.match(ast.STRING):
		return &ast.LiteralPattern{Token: s.previous(), Value: value.NewString(s.previous().Literal)}, nil
	case s.match(ast.BYTES):
		return &ast.LiteralPattern{Token: s.previous(), Value: value.NewBytes([]byte(s.previous().Literal))}, nil
	case s.match(ast.TRUE, ast.FALSE):
		return &ast.LiteralPattern{Token: s.previous(), Value: value.NewBool(s.previous().TokenType == ast.TRUE)}, nil
	case s.match(ast.NIL):
		return &ast.LiteralPattern{Token: s.previous(), Value: value.NilValue}, nil
	case s.match(ast.LEFT_BRACE):
		return s.objectPattern()
	case s.match(ast.LEFT_BRACKET):
		return s.arrayPattern()
	case s.check(ast.MUT):
		b, err := s.binding("Expect variable name.")
		if err != nil {
			return nil, err
		}

		return s.isTail(&ast.BindPattern{Binding: b})
	case s.match(ast.IDENTIFIER):
		name := s.previous()

		if name.Lexeme == "_" {
			return s.isTail(&ast.WildcardPattern{Token: name})
		}

		t, err := s.qualifiedName(name)
		if err != nil {
			return nil, err
		}

		if s.match(ast.LEFT_BRACE) {
			return s.structPattern(t)
		}

		if t.Module != nil {
			return namePattern(t), nil
		}

		return s.isTail(namePattern(t))
	}

	return nil, errors.NewSyntaxError(s.peek(), "Expect pattern.")
}

// namePattern is the pattern of a name: a variant or a constant of a
// module if qualified, else a binding, or a constant the resolver finds.
func namePattern(t *ast.VarExpr) ast.Pattern {
	switch {
	case t.Case != nil:
		return &ast.VariantPattern{Module: t.Module, Enum: t.Name, Variant: *t.Case}
	case t.Module != nil:
		return &ast.VariantPattern{Enum: *t.Module, Variant: t.Name}
	}

	return &ast.BindPattern{Binding: ast.Binding{Name: t.Name}}
}

func (s *Parser) structPattern(t *ast.VarExpr) (ast.Pattern, error) {
	pat := &ast.StructPattern{Type: t}

	var err error
	pat.Fields, _, err = s.fieldPatterns(false)

	return pat, err
}

// objectPattern parses { key: pattern = default, ..., ...rest } after the "{".
func (s *Parser) objectPattern() (ast.Pattern, error) {
	pat := &ast.ObjectPattern{Brace: s.previous()}

	var err error
	pat.Fields, pat.Rest, err = s.fieldPatterns(true)

	if err == nil && len(pat.Fields) == 0 && pat.Rest == nil {
		s.errors = append(s.errors, errors.NewSyntaxError(pat.Brace, "Expect property name in pattern."))
	}

	return pat, err
}

// arrayPattern parses [p, q = default, ...rest] after the "[".
func (s *Parser) arrayPattern() (ast.Pattern, error) {
	pat := &ast.ArrayPattern{Bracket: s.previous()}

	sawDefault := false

	for !s.check(ast.RIGHT_BRACKET) && !s.isAtEnd() {
		if s.match(ast.DOT_DOT_DOT) {
			b, err := s.restBinding(ast.RIGHT_BRACKET)
			if err != nil {
				return nil, err
			}

			pat.Open = true
			if !b.Discards() {
				pat.Rest = &ast.BindPattern{Binding: b}
			}

			break
		}

		start := s.peek()

		p, err := s.matchPattern()
		if err != nil {
			return nil, err
		}

		p, err = s.patternDefault(p)
		if err != nil {
			return nil, err
		}

		_, isDefault := p.(*ast.DefaultPattern)
		if !isDefault && sawDefault {
			s.errors = append(s.errors, errors.NewSyntaxError(start, "An element without a default cannot follow one with a default."))
		}

		sawDefault = sawDefault || isDefault
		pat.Elems = append(pat.Elems, p)

		if !s.match(ast.COMMA) {
			break
		}
	}

	_, err := s.consume(ast.RIGHT_BRACKET, "Expect ']' after pattern.")

	return pat, err
}

// restBinding parses the name after "..." of a rest element, which must
// close its pattern.
func (s *Parser) restBinding(closer ast.TokenType) (ast.Binding, error) {
	b, err := s.binding("Expect variable name.")
	if err != nil {
		return b, err
	}

	if s.check(ast.EQUAL) {
		return b, errors.NewSyntaxError(s.peek(), "A rest element or _ cannot have a default.")
	}

	if !s.check(closer) {
		return b, errors.NewSyntaxError(s.peek(), "Rest element must be last.")
	}

	return b, nil
}

// patternDefault parses the "= value" after p, if any, giving p with its
// default; _ cannot have one.
func (s *Parser) patternDefault(p ast.Pattern) (ast.Pattern, error) {
	if !s.match(ast.EQUAL) {
		return p, nil
	}

	if _, ok := p.(*ast.WildcardPattern); ok {
		return nil, errors.NewSyntaxError(s.previous(), "A rest element or _ cannot have a default.")
	}

	def, err := s.expressionWithStructs(structsAll)

	return &ast.DefaultPattern{Pattern: p, Default: def}, err
}

// isTail parses an optional 'is P' after the pattern p, a name or _.
func (s *Parser) isTail(p ast.Pattern) (ast.Pattern, error) {
	if !s.match(ast.IS) {
		return p, nil
	}

	test, err := s.isPattern()

	return &ast.IsPattern{Inner: p, Test: test}, err
}

// isPattern parses the pattern after 'is', in which a bare name is a type:
// alternatives separated by "|".
func (s *Parser) isPattern() (ast.Pattern, error) {
	return s.alternatives(s.isAlt)
}

func (s *Parser) isAlt() (ast.Pattern, error) {
	tok := s.peek()

	switch t := tok.TokenType; {
	case isTypeKeyword(t):
		return &ast.TypeTestPattern{Token: tok, Type: &ast.VarExpr{Name: s.advance()}}, nil
	case t == ast.IDENTIFIER && s.peek().Lexeme != "_" && s.peekAt(1) != ast.IS:
		v, err := s.qualifiedName(s.advance())
		if err != nil {
			return nil, err
		}

		switch {
		case s.check(ast.DOT_DOT) || s.check(ast.DOT_DOT_EQUAL):
			return s.rangeTail(namePattern(v))
		case s.check(ast.LEFT_BRACE) && s.structPatternAhead():
			s.advance()
			return s.structPattern(v)
		}

		return &ast.TypeTestPattern{Token: tok, Type: v}, nil
	case t == ast.LEFT_PAREN:
		e, err := parseBitXor(s)
		return &ast.TypeTestPattern{Token: tok, Type: e}, err
	}

	return s.patternAlt()
}

// structPatternAhead reports whether the "{" ahead, after a type on the
// right of is, opens the fields of a struct pattern rather than a body
// or the arms of a match: no "=>" or ";" in it, and, where "Name {"
// opens a body, a body or an operator after it.
func (s *Parser) structPatternAhead() bool {
	depth := 0

	for i := 0; ; i++ {
		switch s.peekAt(i) {
		case ast.EOF:
			return false
		case ast.LEFT_BRACE, ast.LEFT_PAREN, ast.LEFT_BRACKET:
			depth++
		case ast.ARROW, ast.SEMICOLON:
			if depth == 1 {
				return false
			}
		case ast.RIGHT_BRACE, ast.RIGHT_PAREN, ast.RIGHT_BRACKET:
			if depth--; depth > 0 {
				continue
			}

			if s.structs == structsAll {
				return true
			}

			switch s.peekAt(i + 1) {
			case ast.LEFT_BRACE, ast.AND, ast.OR, ast.RIGHT_PAREN:
				return true
			}

			return false
		}
	}
}

func (s *Parser) numberPattern(tok ast.Token, negate bool) (ast.Pattern, error) {
	n, _ := helpers.ParseLiteral(tok.Lexeme)

	if negate && n.IsInt() {
		n = value.NewInt(-n.Int())
	} else if negate {
		n = value.NewNumber(-n.Float())
	}

	return &ast.LiteralPattern{Token: tok, Value: n}, nil
}

// fieldPatterns parses key: pattern, ... up to "}"; the "{" has already
// been consumed. A bare key, or mut key, binds the value to its name.
// Objects only have string keys, defaults and a rest, which binds the
// other keys.
func (s *Parser) fieldPatterns(object bool) ([]ast.FieldPattern, *ast.BindPattern, error) {
	var fields []ast.FieldPattern
	var rest *ast.BindPattern

	err := s.braceList("Duplicate property name.", "Expect '}' after pattern.", func() (string, ast.Token, error) {
		if object && s.match(ast.DOT_DOT_DOT) {
			b, err := s.restBinding(ast.RIGHT_BRACE)
			if err == nil && !b.Discards() {
				rest = &ast.BindPattern{Binding: b}
			}

			return "", b.Name, err
		}

		f, err := s.fieldPattern(object)
		if err != nil {
			return "", f.Key, err
		}

		if object {
			f.Pattern, err = s.patternDefault(f.Pattern)
		}

		fields = append(fields, f)

		// A key may repeat, to bind its value and take it apart too.
		return "", f.Key, err
	})

	return fields, rest, err
}

func (s *Parser) fieldPattern(object bool) (ast.FieldPattern, error) {
	if s.check(ast.MUT) {
		b, err := s.binding("Expect property name.")
		return ast.FieldPattern{Key: b.Name, Pattern: &ast.BindPattern{Binding: b}}, err
	}

	if !s.match(ast.IDENTIFIER) && !(object && s.match(ast.STRING)) {
		return ast.FieldPattern{Key: s.peek()}, errors.NewSyntaxError(s.peek(), "Expect property name.")
	}

	key := s.previous()
	quoted := key.TokenType == ast.STRING

	if quoted {
		key.Lexeme = key.Literal
	}

	if !s.match(ast.COLON) {
		if quoted {
			return ast.FieldPattern{Key: key}, errors.NewSyntaxError(s.peek(), "Expect ':' after string key.")
		}

		return ast.FieldPattern{Key: key, Pattern: &ast.BindPattern{Binding: ast.Binding{Name: key}, Shorthand: true}}, nil
	}

	p, err := s.matchPattern()

	return ast.FieldPattern{Key: key, Pattern: p}, err
}
