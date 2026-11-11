package rune

import (
	"strconv"

	"rune/pkg/ast"
	"rune/pkg/errors"
)

func (s *Parser) expression() (ast.Expr, error) {
	return s.assignment()
}

func (s *Parser) assignment() (ast.Expr, error) {
	expr, err := s.rangeExpr()

	if err != nil {
		return nil, err
	}

	if s.match(ast.EQUAL, ast.PLUS_EQUAL, ast.MINUS_EQUAL, ast.STAR_EQUAL, ast.SLASH_EQUAL, ast.PERCENT_EQUAL) {
		equals := s.previous()
		value, err := s.assignment()

		if err != nil {
			return nil, err
		}

		// For "a op= b", op is the binary operator, keeping the "op=" lexeme
		// and line for error messages.
		var op *ast.Token
		if opType, ok := compoundOps[equals.TokenType]; ok {
			t := equals
			t.TokenType = opType
			op = &t
		}

		switch target := expr.(type) {
		case *ast.IndexExpr:
			return ast.NewSetIndexExpr(target, value, op), nil
		case *ast.VarExpr:
			// Reading a variable has no side effects, so "a += b" is just
			// "a = a + b".
			if op != nil {
				value = ast.NewBinaryExpr(ast.NewVarExpr(target.Name), value, *op)
			}

			return ast.NewAssignExpr(target.Name, value), nil
		}

		return nil, errors.NewSyntaxError(equals, "Invalid assignment target.")
	}

	return expr, nil
}

// compoundOps maps each compound assignment operator to its binary operator.
var compoundOps = map[ast.TokenType]ast.TokenType{
	ast.PLUS_EQUAL:    ast.PLUS,
	ast.MINUS_EQUAL:   ast.MINUS,
	ast.STAR_EQUAL:    ast.STAR,
	ast.SLASH_EQUAL:   ast.SLASH,
	ast.PERCENT_EQUAL: ast.PERCENT,
}

func (s *Parser) rangeExpr() (ast.Expr, error) {
	expr, err := s.or()
	if err != nil || !s.match(ast.DOT_DOT, ast.DOT_DOT_EQUAL) {
		return expr, err
	}

	operator := s.previous()

	end, err := s.or()
	if err != nil {
		return nil, err
	}

	var step ast.Expr

	// step is contextual, so it stays usable as a name.
	if s.check(ast.IDENTIFIER) && s.peek().Lexeme == "step" {
		s.advance()

		if step, err = s.or(); err != nil {
			return nil, err
		}
	}

	if s.check(ast.DOT_DOT) || s.check(ast.DOT_DOT_EQUAL) {
		return nil, errors.NewSyntaxError(s.peek(), "Ranges cannot be chained.")
	}

	return ast.NewRangeExpr(expr, end, step, operator), nil
}

func (s *Parser) or() (ast.Expr, error) {
	expr, err := s.and()
	if err != nil {
		return nil, err
	}

	for s.match(ast.OR) {
		operator := s.previous()
		right, err := s.and()
		if err != nil {
			return nil, err
		}

		expr = ast.NewLogicalExpr(expr, right, operator)
	}

	return expr, nil
}

func (s *Parser) and() (ast.Expr, error) {
	expr, err := s.equality()
	if err != nil {
		return nil, err
	}

	for s.match(ast.AND) {
		operator := s.previous()
		right, err := s.equality()
		if err != nil {
			return nil, err
		}

		expr = ast.NewLogicalExpr(expr, right, operator)
	}

	return expr, nil
}

func (s *Parser) equality() (ast.Expr, error) {
	expr, err := s.comparison()

	if err != nil {
		return nil, err
	}

	for s.match(ast.BANG_EQUAL, ast.EQUAL_EQUAL) {
		operator := s.previous()
		right, err := s.comparison()

		if err != nil {
			return nil, err
		}

		expr = ast.NewBinaryExpr(expr, right, operator)
	}

	return expr, nil
}

func (s *Parser) comparison() (ast.Expr, error) {
	expr, err := s.term()

	if err != nil {
		return nil, err
	}

	for s.match(ast.GREATER, ast.GREATER_EQUAL, ast.LESS, ast.LESS_EQUAL) {
		operator := s.previous()
		right, err := s.term()

		if err != nil {
			return nil, err
		}

		expr = ast.NewBinaryExpr(expr, right, operator)
	}

	return expr, nil
}

func (s *Parser) term() (ast.Expr, error) {
	expr, err := s.factor()

	if err != nil {
		return nil, err
	}

	for s.match(ast.PLUS, ast.MINUS) {
		operator := s.previous()
		right, err := s.factor()
		if err != nil {
			return nil, err
		}

		expr = ast.NewBinaryExpr(expr, right, operator)
	}

	return expr, nil
}

func (s *Parser) factor() (ast.Expr, error) {
	expr, err := s.unary()

	if err != nil {
		return nil, err
	}

	for s.match(ast.SLASH, ast.STAR, ast.PERCENT) {
		operator := s.previous()
		right, err := s.unary()

		if err != nil {
			return nil, err
		}

		expr = ast.NewBinaryExpr(expr, right, operator)
	}

	return expr, nil
}

func (s *Parser) unary() (ast.Expr, error) {
	if s.match(ast.BANG, ast.MINUS) {
		operator := s.previous()
		right, err := s.unary()

		if err != nil {
			return nil, err
		}

		return ast.NewUnaryExpr(right, operator), nil
	}

	if s.match(ast.PLUS_PLUS, ast.MINUS_MINUS) {
		operator := s.previous()
		target, err := s.unary()

		if err != nil {
			return nil, err
		}

		return update(target, operator, true)
	}

	return s.postfix()
}

func (s *Parser) postfix() (ast.Expr, error) {
	expr, err := s.call()

	if err != nil {
		return nil, err
	}

	if s.match(ast.PLUS_PLUS, ast.MINUS_MINUS) {
		return update(expr, s.previous(), false)
	}

	return expr, nil
}

func update(target ast.Expr, operator ast.Token, prefix bool) (ast.Expr, error) {
	switch target.(type) {
	case *ast.VarExpr, *ast.IndexExpr:
		return ast.NewUpdateExpr(target, operator, prefix), nil
	}

	return nil, errors.NewSyntaxError(operator, "Invalid increment target.")
}

func (s *Parser) call() (ast.Expr, error) {
	expr, err := s.primary()
	if err != nil {
		return nil, err
	}

	for {
		if s.match(ast.LEFT_PAREN) {
			expr, err = s.finishCall(expr)

			if err != nil {
				return nil, err
			}
		} else if s.match(ast.LEFT_BRACKET) {
			index, err := s.expression()
			if err != nil {
				return nil, err
			}

			_, err = s.consume(ast.RIGHT_BRACKET, "Expect ']' after index.")
			if err != nil {
				return nil, err
			}

			expr = ast.NewIndexExpr(expr, index, s.previous())
		} else if s.match(ast.DOT) {
			// obj.name is sugar for obj["name"].
			name, err := s.consume(ast.IDENTIFIER, "Expect property name after '.'.")
			if err != nil {
				return nil, err
			}

			expr = ast.NewPropertyExpr(expr, name)
		} else {
			break
		}
	}

	return expr, nil
}

func (s *Parser) finishCall(expr ast.Expr) (ast.Expr, error) {
	args := []ast.Expr{}

	if !s.check(ast.RIGHT_PAREN) {
		for {
			arg, err := s.expression()
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

	return ast.NewCallExpr(s.previous(), expr, args), nil
}

func (s *Parser) primary() (ast.Expr, error) {
	if s.match(ast.LEFT_BRACE) {
		return s.objectLiteral()
	}

	if s.match(ast.LEFT_BRACKET) {
		return s.arrayLiteral()
	}

	if s.check(ast.FUN) && s.peekAt(1) == ast.LEFT_PAREN {
		return s.functionExpr()
	}

	if s.match(ast.TRUE) {
		return ast.NewLiteralExpr(ast.TRUE, true), nil
	}

	if s.match(ast.FALSE) {
		return ast.NewLiteralExpr(ast.FALSE, false), nil
	}

	if s.match(ast.NIL) {
		return ast.NewLiteralExpr(ast.NIL, nil), nil
	}

	if s.match(ast.NUMBER) {
		prev := s.previous()
		value, err := strconv.ParseFloat(prev.Lexeme, 64)

		if err != nil {
			return nil, errors.NewSyntaxError(prev, "Invalid number.")
		}

		return ast.NewLiteralExpr(ast.NUMBER, value), nil
	}

	if s.match(ast.STRING) {
		prev := s.previous()

		return ast.NewLiteralExpr(ast.STRING, prev.Literal), nil
	}

	if s.match(ast.IDENTIFIER) {
		prev := s.previous()

		if s.isStructLiteralStart() {
			s.advance()
			return s.structLiteral(prev)
		}

		return ast.NewVarExpr(prev), nil
	}

	if s.match(ast.LEFT_PAREN) {
		expr, err := s.expression()

		if err != nil {
			return nil, err
		}

		_, err = s.consume(ast.RIGHT_PAREN, "Expect ')' after expression.")
		if err != nil {
			return nil, err
		}

		return ast.NewGroupingExpr(expr), nil
	}

	return nil, errors.NewSyntaxError(s.peek(), "Expect expression.")
}

// braceList parses comma-separated items up to "}", a trailing comma
// allowed; the "{" has already been consumed. item parses one and returns
// the name it gives, reported as dup when it repeats.
func (s *Parser) braceList(dup, closeMsg string, item func() (string, ast.Token, error)) error {
	seen := map[string]bool{}

	for !s.check(ast.RIGHT_BRACE) && !s.isAtEnd() {
		name, tok, err := item()
		if err != nil {
			return err
		}

		// Report but keep parsing: the parser is not in a confused state.
		if seen[name] {
			s.errors = append(s.errors, errors.NewSyntaxError(tok, dup))
		}

		seen[name] = true

		if !s.match(ast.COMMA) {
			break
		}
	}

	_, err := s.consume(ast.RIGHT_BRACE, closeMsg)

	return err
}

// objectLiteral parses { key: value, ... }; the "{" has already been consumed.
func (s *Parser) objectLiteral() (ast.Expr, error) {
	var pairs []ast.ObjectPair

	err := s.braceList("Duplicate property name.", "Expect '}' after object properties.", func() (string, ast.Token, error) {
		var key string

		switch {
		case s.match(ast.IDENTIFIER):
			key = s.previous().Lexeme
		case s.match(ast.STRING):
			key = s.previous().Literal
		default:
			return "", ast.Token{}, errors.NewSyntaxError(s.peek(), "Expect property name.")
		}

		tok := s.previous()

		if _, err := s.consume(ast.COLON, "Expect ':' after property name."); err != nil {
			return "", tok, err
		}

		v, err := s.expression()
		pairs = append(pairs, ast.ObjectPair{Key: key, Value: v})

		return key, tok, err
	})
	if err != nil {
		return nil, err
	}

	return ast.NewObjectExpr(ast.OBJECT, pairs), nil
}

// structLiteral parses Name { field: value, ... }; "Name {" has already been consumed.
func (s *Parser) structLiteral(name ast.Token) (ast.Expr, error) {
	var pairs []ast.StructPair

	err := s.braceList("Duplicate field name.", "Expect '}' after struct fields.", func() (string, ast.Token, error) {
		key, err := s.consume(ast.IDENTIFIER, "Expect field name.")
		if err != nil {
			return "", key, err
		}

		if _, err := s.consume(ast.COLON, "Expect ':' after field name."); err != nil {
			return "", key, err
		}

		v, err := s.expression()
		pairs = append(pairs, ast.StructPair{Key: key, Value: v})

		return key.Lexeme, key, err
	})
	if err != nil {
		return nil, err
	}

	return ast.NewStructExpr(name, pairs), nil
}

// isStructLiteralStart reports whether a name is followed by "{ }" or "{ field :".
func (s *Parser) isStructLiteralStart() bool {
	if !s.check(ast.LEFT_BRACE) {
		return false
	}

	next := s.peekAt(1)

	return next == ast.RIGHT_BRACE || next == ast.IDENTIFIER && s.peekAt(2) == ast.COLON
}

// arrayLiteral parses [a, b, ...]; the "[" has already been consumed.
func (s *Parser) arrayLiteral() (ast.Expr, error) {
	var items []ast.Expr

	if !s.check(ast.RIGHT_BRACKET) {
		for {
			elem, err := s.expression()
			if err != nil {
				return nil, err
			}

			items = append(items, elem)

			// Allow a trailing comma, as in object literals.
			if !s.match(ast.COMMA) || s.check(ast.RIGHT_BRACKET) {
				break
			}
		}
	}

	if _, err := s.consume(ast.RIGHT_BRACKET, "Expect ']' after array elements."); err != nil {
		return nil, err
	}

	return ast.NewArrayExpr(ast.ARRAY, items), nil
}

// functionExpr parses an anonymous function: fun (params) { body }.
func (s *Parser) functionExpr() (ast.Expr, error) {
	keyword := s.advance()
	s.advance()

	name := ast.NewToken(ast.FUN, "", "", keyword.Line)
	name.Col = keyword.Col

	decl, err := s.functionRest(name)
	if err != nil {
		return nil, err
	}

	return ast.NewFunctionExpr(decl), nil
}

// isObjectLiteralStart reports whether the upcoming tokens are "{", a property name and ":".
func (s *Parser) isObjectLiteralStart() bool {
	if !s.check(ast.LEFT_BRACE) || s.current+2 >= len(s.tokens) {
		return false
	}

	key := s.tokens[s.current+1].TokenType

	return (key == ast.IDENTIFIER || key == ast.STRING) && s.tokens[s.current+2].TokenType == ast.COLON
}
