package rune

import (
	"rune/pkg/ast"
	"rune/pkg/errors"
)

func (s *Parser) call() (ast.Expr, error) {
	expr, err := s.primary()
	if err != nil {
		return nil, err
	}

	chain := false

	for {
		optional := s.match(ast.QUESTION_DOT)
		chain = chain || optional

		switch {
		case s.match(ast.LEFT_PAREN):
			expr, err = s.finishCall(expr, optional)
		case s.match(ast.LEFT_BRACKET):
			expr, err = s.finishIndex(expr, optional)
		case optional || s.match(ast.DOT):
			expr, err = s.property(expr, optional)
		case chain:
			return &ast.OptionalExpr{Expr: expr}, nil
		default:
			return expr, nil
		}

		if err != nil {
			return nil, err
		}
	}
}

// finishIndex parses the rest of arr[i], or of a slice arr[a..b], after "[".
func (s *Parser) finishIndex(expr ast.Expr, optional bool) (ast.Expr, error) {
	defer s.allowStructs(structsAll)()

	var index ast.Expr
	var err error

	if s.match(ast.DOT_DOT, ast.DOT_DOT_EQUAL) {
		index, err = s.rangeAfter(nil, s.previous())
	} else {
		index, err = s.expression()
	}

	if err != nil {
		return nil, err
	}

	r, isSlice := index.(*ast.RangeExpr)
	if isSlice && r.Step != nil {
		return nil, errors.NewSyntaxError(r.Operator, "A slice takes no step.")
	}

	if _, err = s.consume(ast.RIGHT_BRACKET, "Expect ']' after index."); err != nil {
		return nil, err
	}

	if isSlice {
		return &ast.SliceExpr{Target: expr, Range: r, Optional: optional}, nil
	}

	return &ast.IndexExpr{Array: expr, Index: index, Token: s.previous(), Optional: optional}, nil
}

// property parses the name of obj.name, sugar for obj["name"], after the dot.
func (s *Parser) property(expr ast.Expr, optional bool) (ast.Expr, error) {
	name, err := s.consume(ast.IDENTIFIER, "Expect property name after '"+s.previous().Lexeme+"'.")
	if err != nil {
		return nil, err
	}

	return ast.NewPropertyExpr(expr, name, optional), nil
}

// item parses an element of a list, or ...expr if spread allows it.
func (s *Parser) item(spread bool) (ast.Expr, error) {
	if !spread || !s.match(ast.DOT_DOT_DOT) {
		return s.expressionWithStructs(structsAll)
	}

	tok := s.previous()

	e, err := s.expressionWithStructs(structsAll)
	if err != nil {
		return nil, err
	}

	return &ast.SpreadExpr{Token: tok, Expr: e}, nil
}

func (s *Parser) finishCall(expr ast.Expr, optional bool) (ast.Expr, error) {
	args := []ast.Expr{}

	if !s.check(ast.RIGHT_PAREN) {
		for {
			arg, err := s.item(true)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if !s.match(ast.COMMA) {
				break
			}
		}
	}

	_, err := s.consume(ast.RIGHT_PAREN, "Expect ')' after arguments.")
	if err != nil {
		return nil, err
	}

	return &ast.CallExpr{Token: s.previous(), Callee: expr, Args: args, Optional: optional}, nil
}
