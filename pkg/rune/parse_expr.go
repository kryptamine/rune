package rune

import (
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

func (s *Parser) expression() (ast.Expr, error) {
	return s.assignment()
}

func (s *Parser) assignment() (ast.Expr, error) {
	if fn, ok, err := s.arrowFunction(); ok {
		return fn, err
	}

	expr, err := s.conditional()

	if err != nil {
		return nil, err
	}

	if s.match(ast.EQUAL, ast.PLUS_EQUAL, ast.MINUS_EQUAL, ast.STAR_EQUAL, ast.SLASH_EQUAL, ast.PERCENT_EQUAL,
		ast.AMP_EQUAL, ast.PIPE_EQUAL, ast.CARET_EQUAL, ast.LESS_LESS_EQUAL, ast.GREATER_GREATER_EQUAL) {
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

	ast.AMP_EQUAL:             ast.AMP,
	ast.PIPE_EQUAL:            ast.PIPE,
	ast.CARET_EQUAL:           ast.CARET,
	ast.LESS_LESS_EQUAL:       ast.LESS_LESS,
	ast.GREATER_GREATER_EQUAL: ast.GREATER_GREATER,
}

// conditional parses cond ? then : else, right-associative.
func (s *Parser) conditional() (ast.Expr, error) {
	cond, err := s.rangeExpr()
	if err != nil || !s.match(ast.QUESTION) {
		return cond, err
	}

	then, err := s.expressionWithStructs(structsAll)
	if err != nil {
		return nil, err
	}

	if _, err := s.consume(ast.COLON, "Expect ':' after then branch of '?'."); err != nil {
		return nil, err
	}

	els, err := s.assignment()
	if err != nil {
		return nil, err
	}

	return &ast.ConditionalExpr{Cond: cond, Then: then, Else: els}, nil
}

func (s *Parser) rangeExpr() (ast.Expr, error) {
	expr, err := s.nullish()
	if err != nil || !s.match(ast.DOT_DOT, ast.DOT_DOT_EQUAL) {
		return expr, err
	}

	operator := s.previous()

	end, err := s.nullish()
	if err != nil {
		return nil, err
	}

	var step ast.Expr

	// step is contextual, so it stays usable as a name.
	if s.check(ast.IDENTIFIER) && s.peek().Lexeme == "step" {
		s.advance()

		if step, err = s.nullish(); err != nil {
			return nil, err
		}
	}

	if s.check(ast.DOT_DOT) || s.check(ast.DOT_DOT_EQUAL) {
		return nil, errors.NewSyntaxError(s.peek(), "Ranges cannot be chained.")
	}

	return ast.NewRangeExpr(expr, end, step, operator), nil
}

// nullish parses a ?? b, which is b only if a is nil.
func (s *Parser) nullish() (ast.Expr, error) {
	expr, err := s.or()

	for err == nil && s.match(ast.QUESTION_QUESTION) {
		operator := s.previous()

		var right ast.Expr
		if right, err = s.or(); err == nil {
			expr = ast.NewLogicalExpr(expr, right, operator)
		}
	}

	return expr, err
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

	for s.match(ast.BANG_EQUAL, ast.EQUAL_EQUAL, ast.IS) {
		operator := s.previous()
		right, err := s.comparison()

		if err != nil {
			return nil, err
		}

		if operator.TokenType == ast.IS {
			expr = ast.NewIsExpr(expr, right, operator)
		} else {
			expr = ast.NewBinaryExpr(expr, right, operator)
		}
	}

	return expr, nil
}

func (s *Parser) comparison() (ast.Expr, error) {
	return s.binaryLevel(parseBitOr, ast.GREATER, ast.GREATER_EQUAL, ast.LESS, ast.LESS_EQUAL)
}

// The bitwise levels bind tighter than comparison and looser than +, as
// in Python: | then ^ then & then shifts.
func parseBitOr(s *Parser) (ast.Expr, error) { return s.binaryLevel(parseBitXor, ast.PIPE) }

func parseBitXor(s *Parser) (ast.Expr, error) { return s.binaryLevel(parseBitAnd, ast.CARET) }

func parseBitAnd(s *Parser) (ast.Expr, error) { return s.binaryLevel(parseShift, ast.AMP) }

func parseShift(s *Parser) (ast.Expr, error) {
	return s.binaryLevel((*Parser).term, ast.LESS_LESS, ast.GREATER_GREATER)
}

// binaryLevel parses a left-associative chain of operands joined by ops.
func (s *Parser) binaryLevel(operand func(*Parser) (ast.Expr, error), ops ...ast.TokenType) (ast.Expr, error) {
	expr, err := operand(s)
	if err != nil {
		return nil, err
	}

	for s.match(ops...) {
		operator := s.previous()

		right, err := operand(s)
		if err != nil {
			return nil, err
		}

		expr = ast.NewBinaryExpr(expr, right, operator)
	}

	return expr, nil
}

func (s *Parser) term() (ast.Expr, error) {
	return s.binaryLevel((*Parser).factor, ast.PLUS, ast.MINUS)
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
	if s.match(ast.THROW) {
		keyword := s.previous()

		v, err := s.assignment()
		if err != nil {
			return nil, err
		}

		return &ast.ThrowExpr{Value: v, Keyword: keyword}, nil
	}

	if s.match(ast.BANG, ast.MINUS, ast.TILDE) {
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

// allowStructs sets which "Name {" start a struct literal and returns a
// func restoring the previous setting.
func (s *Parser) allowStructs(mode structMode) (restore func()) {
	prev := s.structs
	s.structs = mode

	return func() { s.structs = prev }
}

// expressionWithStructs parses an expression with struct literals allowed
// as mode says: all again inside brackets.
func (s *Parser) expressionWithStructs(mode structMode) (ast.Expr, error) {
	defer s.allowStructs(mode)()

	return s.expression()
}

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

// finishIndex parses the rest of arr[i] after "[".
func (s *Parser) finishIndex(expr ast.Expr, optional bool) (ast.Expr, error) {
	index, err := s.expressionWithStructs(structsAll)
	if err != nil {
		return nil, err
	}

	if _, err = s.consume(ast.RIGHT_BRACKET, "Expect ']' after index."); err != nil {
		return nil, err
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

	if s.match(ast.MATCH) {
		return s.matchExpr()
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
		v, _ := helpers.ParseLiteral(s.previous().Lexeme)

		return ast.NewLiteralExpr(ast.NUMBER, v), nil
	}

	if s.match(ast.STRING) {
		prev := s.previous()

		return ast.NewLiteralExpr(ast.STRING, prev.Literal), nil
	}

	if s.match(ast.INTERPOLATION) {
		return s.interpolation()
	}

	if s.match(ast.BYTES) {
		return ast.NewLiteralExpr(ast.BYTES, value.NewBytes([]byte(s.previous().Literal))), nil
	}

	if s.match(ast.IDENTIFIER) {
		prev := s.previous()

		if s.isStructLiteralStart(0) {
			s.advance()
			return s.structLiteral(nil, prev)
		}

		if s.check(ast.DOT) && s.peekAt(1) == ast.IDENTIFIER && s.isStructLiteralStart(2) {
			s.advance()
			name := s.advance()
			s.advance()

			return s.structLiteral(&prev, name)
		}

		return ast.NewVarExpr(prev), nil
	}

	if s.match(ast.LEFT_PAREN) {
		expr, err := s.expressionWithStructs(structsAll)

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
		// A spread has no name.
		if seen[name] && tok.TokenType != ast.DOT_DOT_DOT {
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

// objectLiteral parses { key: value, ... }; the "{" has already been
// consumed. A bare name takes the variable of the same name.
func (s *Parser) objectLiteral() (ast.Expr, error) {
	var pairs []ast.ObjectPair

	err := s.braceList("Duplicate property name.", "Expect '}' after object properties.", func() (string, ast.Token, error) {
		var key string

		if s.check(ast.DOT_DOT_DOT) {
			tok := s.peek()
			v, err := s.item(true)
			pairs = append(pairs, ast.ObjectPair{Value: v})

			return "", tok, err
		}

		switch {
		case s.match(ast.IDENTIFIER):
			key = s.previous().Lexeme
		case s.match(ast.STRING):
			key = s.previous().Literal
		default:
			return "", ast.Token{}, errors.NewSyntaxError(s.peek(), "Expect property name.")
		}

		tok := s.previous()

		if tok.TokenType == ast.IDENTIFIER && endsItem(s.peekAt(0)) {
			pairs = append(pairs, ast.ObjectPair{Key: key, Value: ast.NewVarExpr(tok)})
			return key, tok, nil
		}

		if _, err := s.consume(ast.COLON, "Expect ':' after property name."); err != nil {
			return "", tok, err
		}

		v, err := s.expressionWithStructs(structsAll)
		pairs = append(pairs, ast.ObjectPair{Key: key, Value: v})

		return key, tok, err
	})
	if err != nil {
		return nil, err
	}

	return ast.NewObjectExpr(ast.OBJECT, pairs), nil
}

// structLiteral parses [module.]Name { field: value, ... }; "Name {" has already been consumed.
func (s *Parser) structLiteral(module *ast.Token, name ast.Token) (ast.Expr, error) {
	pairs, err := s.structPairs()
	if err != nil {
		return nil, err
	}

	return &ast.StructExpr{Type: &ast.VarExpr{Module: module, Name: name}, Pairs: pairs}, nil
}

// structPairs parses field: value, ... up to "}"; the "{" has already been
// consumed. A bare field takes the variable of the same name.
func (s *Parser) structPairs() ([]ast.StructPair, error) {
	var pairs []ast.StructPair

	err := s.braceList("Duplicate field name.", "Expect '}' after struct fields.", func() (string, ast.Token, error) {
		if s.check(ast.DOT_DOT_DOT) {
			tok := s.peek()
			v, err := s.item(true)
			pairs = append(pairs, ast.StructPair{Key: tok, Value: v})

			return "", tok, err
		}

		key, err := s.consume(ast.IDENTIFIER, "Expect field name.")
		if err != nil {
			return "", key, err
		}

		if !s.match(ast.COLON) {
			pairs = append(pairs, ast.StructPair{Key: key, Value: ast.NewVarExpr(key)})

			return key.Lexeme, key, nil
		}

		v, err := s.expressionWithStructs(structsAll)
		pairs = append(pairs, ast.StructPair{Key: key, Value: v})

		return key.Lexeme, key, err
	})

	return pairs, err
}

// isStructLiteralStart reports whether "Name {" at offset starts a struct literal.
func (s *Parser) isStructLiteralStart(offset int) bool {
	if s.structs == structsNone || s.peekAt(offset) != ast.LEFT_BRACE {
		return false
	}

	switch s.peekAt(offset + 1) {
	case ast.RIGHT_BRACE:
		return s.structs == structsAll
	case ast.DOT_DOT_DOT:
		return true
	case ast.IDENTIFIER:
		after := s.peekAt(offset + 2)
		return after == ast.COLON || after == ast.COMMA || after == ast.RIGHT_BRACE
	}

	return false
}

// arrayLiteral parses [a, b, ...]; the "[" has already been consumed.
func (s *Parser) arrayLiteral() (ast.Expr, error) {
	var items []ast.Expr

	if !s.check(ast.RIGHT_BRACKET) {
		for {
			elem, err := s.item(true)
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
	defer s.allowStructs(structsAll)()

	name := anonName(s.advance())
	s.advance()

	decl, err := s.functionRest(name)
	if err != nil {
		return nil, err
	}

	return ast.NewFunctionExpr(decl), nil
}

// arrowFunction parses (params) => body or param => body; ok is false,
// with nothing consumed, if the tokens ahead are not one.
func (s *Parser) arrowFunction() (fn ast.Expr, ok bool, err error) {
	start, nerrs := s.current, len(s.errors)
	name := anonName(s.peek())

	params, ok := s.arrowParams()
	if !ok || !s.check(ast.ARROW) {
		s.current, s.errors = start, s.errors[:nerrs]
		return nil, false, nil
	}

	body, err := s.arrowBody(s.advance())
	if err != nil {
		return nil, true, err
	}

	return ast.NewFunctionExpr(params.function(name, body)), true, nil
}

func (s *Parser) arrowParams() (params, bool) {
	if s.match(ast.LEFT_PAREN) {
		ps, err := s.parameters()
		return ps, err == nil
	}

	if s.bindingEnd(0) < 0 {
		return params{}, false
	}

	b, _ := s.binding("")

	return params{list: []ast.Binding{b}, defaults: []ast.Expr{nil}}, true
}

// arrowBody parses a block, or an expression whose value is returned.
func (s *Parser) arrowBody(arrow ast.Token) ([]ast.Stmt, error) {
	if body, ok, err := s.braceBody(); ok {
		return body, err
	}

	e, err := s.assignment()

	return []ast.Stmt{ast.NewReturnStmt(e, arrow)}, err
}

func anonName(at ast.Token) ast.Token {
	name := ast.NewToken(ast.FUN, "", "", at.Line)
	name.Col = at.Col

	return name
}

// isObjectLiteralStart reports whether the upcoming tokens are "{" and
// a property name and ":", or "...".
func (s *Parser) isObjectLiteralStart() bool {
	if !s.check(ast.LEFT_BRACE) {
		return false
	}

	key, after := s.peekAt(1), s.peekAt(2)

	return key == ast.DOT_DOT_DOT ||
		(key == ast.IDENTIFIER || key == ast.STRING) && after == ast.COLON ||
		key == ast.IDENTIFIER && endsItem(after)
}

// endsItem reports whether t ends an item of a brace list, as after a
// shorthand property.
func endsItem(t ast.TokenType) bool { return t == ast.COMMA || t == ast.RIGHT_BRACE }

// interpolation parses the pieces of a string after its first one,
// an INTERPOLATION token.
func (s *Parser) interpolation() (ast.Expr, error) {
	var parts []ast.Expr

	for {
		piece := s.previous()
		if piece.Literal != "" {
			parts = append(parts, ast.NewLiteralExpr(ast.STRING, piece.Literal))
		}

		if piece.TokenType == ast.STRING_END {
			return &ast.InterpolationExpr{Parts: parts}, nil
		}

		expr, err := s.expressionWithStructs(structsAll)
		if err != nil {
			return nil, err
		}

		parts = append(parts, expr)

		if !s.match(ast.INTERPOLATION, ast.STRING_END) {
			return nil, errors.NewSyntaxError(s.peek(), "Expect '}' after interpolated expression.")
		}
	}
}
