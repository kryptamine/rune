package rune

import (
	"rune/pkg/ast"
	"rune/pkg/errors"
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

		if s.check(ast.SEMICOLON) {
			return nil, errors.NewSyntaxError(s.peek(), "Separate match arms with ','.")
		}

		if !s.match(ast.COMMA) {
			break
		}
	}

	if len(arms) == 0 {
		s.errors = append(s.errors, errors.NewSyntaxError(s.peek(), "Expect match arm."))
	}

	if _, err := s.consume(ast.RIGHT_BRACE, "Expect '}' after match arms."); err != nil {
		return nil, err
	}

	m := &ast.MatchExpr{Keyword: keyword, Subject: subject, Arms: arms}

	if is, ok := subject.(*ast.IsExpr); ok && !is.Negate {
		if t, ok := is.Pattern.(*ast.TypeTestPattern); ok {
			if v, ok := t.Type.(*ast.VarExpr); ok {
				m.Subject, m.Enum = is.Value, v
			}
		}
	}

	return m, nil
}

func (s *Parser) matchArm() (ast.MatchArm, error) {
	arm := ast.MatchArm{Token: s.peek()}

	var err error

	if arm.Pattern, err = s.matchPattern(); err != nil {
		return arm, err
	}

	if s.match(ast.IF) {
		if arm.Guard, err = s.nullish(); err != nil {
			return arm, err
		}
	}

	if _, err := s.consume(ast.ARROW, "Expect '=>' after pattern."); err != nil {
		return arm, err
	}

	arm.Body, err = s.armBody()

	return arm, err
}

// armBody parses a block or an expression.
func (s *Parser) armBody() (*ast.BlockStmt, error) {
	if s.check(ast.LEFT_BRACE) {
		return s.braced(true)
	}

	e, err := s.expression()

	return &ast.BlockStmt{Stmts: []ast.Stmt{ast.NewExprStmt(e)}, Valued: true}, err
}
