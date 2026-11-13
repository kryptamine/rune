package rune

import (
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

// matchExpr parses the rest of match subject { arms } after "match". The
// subject may be a struct literal: an arm cannot start with "name :".
func (s *Parser) matchExpr() (*ast.MatchExpr, error) {
	keyword := s.previous()

	subject, err := s.expressionWithStructs(structsNonEmpty)
	if err != nil {
		return nil, err
	}

	defer s.allowStructs(structsAll)()

	if _, err := s.consume(ast.LEFT_BRACE, "Expect '{' after match value."); err != nil {
		return nil, err
	}

	var arms []ast.MatchArm

	for !s.check(ast.RIGHT_BRACE) && !s.isAtEnd() {
		arm, err := s.matchArm()
		if err != nil {
			return nil, err
		}

		arms = append(arms, arm)

		// Only an expression needs a comma, or a semicolon, to end it.
		if _, isExpr := arm.Body.(*ast.ExprStmt); !s.match(ast.COMMA) && isExpr && !s.match(ast.SEMICOLON) {
			break
		}
	}

	if len(arms) == 0 {
		s.errors = append(s.errors, errors.NewSyntaxError(s.peek(), "Expect match arm."))
	}

	if _, err := s.consume(ast.RIGHT_BRACE, "Expect '}' after match arms."); err != nil {
		return nil, err
	}

	return &ast.MatchExpr{Keyword: keyword, Subject: subject, Arms: arms}, nil
}

func (s *Parser) matchArm() (ast.MatchArm, error) {
	arm := ast.MatchArm{Token: s.peek()}

	var err error

	if arm.Pattern, err = s.matchPattern(); err != nil {
		return arm, err
	}

	if s.match(ast.IF) {
		if arm.Guard, err = s.rangeExpr(); err != nil {
			return arm, err
		}
	}

	if _, err := s.consume(ast.ARROW, "Expect '=>' after pattern."); err != nil {
		return arm, err
	}

	arm.Body, err = s.armBody()

	return arm, err
}

// armBody parses a statement starting with a keyword, a block, or an
// expression, which the arm's comma ends.
func (s *Parser) armBody() (ast.Stmt, error) {
	if t := s.peek().TokenType; stmtKeyword(t) && t != ast.THROW {
		return s.statement()
	}

	if stmts, ok, err := s.braceBody(); ok {
		return ast.NewBlockStmt(stmts), err
	}

	e, err := s.expression()

	return ast.NewExprStmt(e), err
}

// matchPattern parses a pattern: alternatives separated by "|".
func (s *Parser) matchPattern() (ast.Pattern, error) {
	p, err := s.patternAlt()
	if err != nil || !s.check(ast.PIPE) {
		return p, err
	}

	alts := []ast.Pattern{p}

	for s.match(ast.PIPE) {
		alt, err := s.patternAlt()
		if err != nil {
			return nil, err
		}

		alts = append(alts, alt)
	}

	return &ast.OrPattern{Alts: alts}, nil
}

func (s *Parser) patternAlt() (ast.Pattern, error) {
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
		brace := s.previous()

		fields, err := s.fieldPatterns(true)
		if err == nil && len(fields) == 0 {
			s.errors = append(s.errors, errors.NewSyntaxError(brace, "Expect property name in pattern."))
		}

		return &ast.ObjectPattern{Brace: brace, Fields: fields}, err
	case s.check(ast.MUT):
		b, err := s.binding("Expect variable name.")
		if err != nil {
			return nil, err
		}

		return s.typePattern(&ast.BindPattern{Binding: b})
	case s.match(ast.IDENTIFIER):
		name := s.previous()

		if name.Lexeme == "_" {
			return s.typePattern(&ast.WildcardPattern{Token: name})
		}

		pat := &ast.StructPattern{Name: name}

		if s.match(ast.DOT) {
			typ, err := s.consume(ast.IDENTIFIER, "Expect struct name after '.'.")
			if err != nil {
				return nil, err
			}

			pat.Module, pat.Name = &name, typ
		}

		if pat.Module != nil && s.match(ast.DOT) {
			variant, err := s.consume(ast.IDENTIFIER, "Expect variant name after '.'.")
			return &ast.VariantPattern{Module: pat.Module, Enum: pat.Name, Variant: variant}, err
		}

		if s.match(ast.LEFT_BRACE) {
			fields, err := s.fieldPatterns(false)
			pat.Fields = fields

			return pat, err
		}

		if pat.Module != nil {
			return &ast.VariantPattern{Enum: name, Variant: pat.Name}, nil
		}

		return s.typePattern(&ast.BindPattern{Binding: ast.Binding{Name: name}})
	}

	return nil, errors.NewSyntaxError(s.peek(), "Expect pattern.")
}

// typePattern parses an optional 'is T | U' after the pattern p, a name or _.
func (s *Parser) typePattern(p ast.Pattern) (ast.Pattern, error) {
	if !s.match(ast.IS) {
		return p, nil
	}

	types, err := s.typeList()

	return &ast.TypePattern{Inner: p, Types: types}, err
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
// been consumed. A bare key, or mut key, binds the value to its name;
// string keys are for objects only.
func (s *Parser) fieldPatterns(object bool) ([]ast.FieldPattern, error) {
	var fields []ast.FieldPattern

	err := s.braceList("Duplicate property name.", "Expect '}' after pattern.", func() (string, ast.Token, error) {
		if s.check(ast.MUT) {
			b, err := s.binding("Expect property name.")
			fields = append(fields, ast.FieldPattern{Key: b.Name, Pattern: &ast.BindPattern{Binding: b}})

			return b.Name.Lexeme, b.Name, err
		}

		if !s.match(ast.IDENTIFIER) && !(object && s.match(ast.STRING)) {
			return "", s.peek(), errors.NewSyntaxError(s.peek(), "Expect property name.")
		}

		key := s.previous()
		quoted := key.TokenType == ast.STRING

		if quoted {
			key.Lexeme = key.Literal
		}

		if !s.match(ast.COLON) {
			if quoted {
				return "", key, errors.NewSyntaxError(s.peek(), "Expect ':' after string key.")
			}

			fields = append(fields, ast.FieldPattern{Key: key, Pattern: &ast.BindPattern{Binding: ast.Binding{Name: key}}})

			return key.Lexeme, key, nil
		}

		p, err := s.matchPattern()
		fields = append(fields, ast.FieldPattern{Key: key, Pattern: p})

		return key.Lexeme, key, err
	})

	return fields, err
}
