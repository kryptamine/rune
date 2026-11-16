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

	expr, err := s.nullish()

	if err != nil {
		return nil, err
	}

	if s.match(ast.EQUAL, ast.PLUS_EQUAL, ast.MINUS_EQUAL, ast.STAR_EQUAL, ast.SLASH_EQUAL, ast.PERCENT_EQUAL,
		ast.AMP_EQUAL, ast.PIPE_EQUAL, ast.CARET_EQUAL, ast.LESS_LESS_EQUAL, ast.GREATER_GREATER_EQUAL,
		ast.GREATER_GREATER_GREATER_EQUAL) {
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

	ast.GREATER_GREATER_GREATER_EQUAL: ast.GREATER_GREATER_GREATER,
}

// rangeExpr parses a..b and a..=b, which bind looser than the bitwise and
// arithmetic operators and tighter than comparisons.
func (s *Parser) rangeExpr() (ast.Expr, error) {
	expr, err := parseBitOr(s)
	if err != nil || !s.match(ast.DOT_DOT, ast.DOT_DOT_EQUAL) {
		return expr, err
	}

	return s.rangeAfter(expr, s.previous())
}

// rangeAfter parses the rest of a range after its operator. Its end may
// be left open before "]", for a slice; the resolver rejects it elsewhere.
func (s *Parser) rangeAfter(start ast.Expr, operator ast.Token) (ast.Expr, error) {
	if s.check(ast.RIGHT_BRACKET) && operator.TokenType == ast.DOT_DOT {
		return ast.NewRangeExpr(start, nil, nil, operator), nil
	}

	return s.rangeRest(start, operator)
}

// rangeRest parses the end and step of a range after its operator.
func (s *Parser) rangeRest(start ast.Expr, operator ast.Token) (ast.Expr, error) {
	end, err := parseBitOr(s)
	if err != nil {
		return nil, err
	}

	var step ast.Expr

	// step is contextual, so it stays usable as a name.
	if s.check(ast.IDENTIFIER) && s.peek().Lexeme == "step" {
		s.advance()

		if step, err = parseBitOr(s); err != nil {
			return nil, err
		}
	}

	if s.check(ast.DOT_DOT) || s.check(ast.DOT_DOT_EQUAL) {
		return nil, errors.NewSyntaxError(s.peek(), "Ranges cannot be chained.")
	}

	return ast.NewRangeExpr(start, end, step, operator), nil
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

	for s.match(ast.BANG_EQUAL, ast.EQUAL_EQUAL, ast.IS, ast.BANG_IS, ast.IN, ast.BANG_IN) {
		operator := s.previous()

		if t := operator.TokenType; t == ast.IS || t == ast.BANG_IS {
			pat, err := s.isPattern()
			if err != nil {
				return nil, err
			}

			expr = &ast.IsExpr{Value: expr, Pattern: pat, Token: operator, Negate: t == ast.BANG_IS}

			continue
		}

		right, err := s.comparison()
		if err != nil {
			return nil, err
		}

		expr = ast.NewBinaryExpr(expr, right, operator)
	}

	return expr, nil
}

func (s *Parser) comparison() (ast.Expr, error) {
	return s.binaryLevel((*Parser).rangeExpr, ast.GREATER, ast.GREATER_EQUAL, ast.LESS, ast.LESS_EQUAL)
}

// The bitwise levels bind tighter than comparison and looser than +, as
// in Python: | then ^ then & then shifts.
func parseBitOr(s *Parser) (ast.Expr, error) { return s.binaryLevel(parseBitXor, ast.PIPE) }

func parseBitXor(s *Parser) (ast.Expr, error) { return s.binaryLevel(parseBitAnd, ast.CARET) }

func parseBitAnd(s *Parser) (ast.Expr, error) { return s.binaryLevel(parseShift, ast.AMP) }

func parseShift(s *Parser) (ast.Expr, error) {
	return s.binaryLevel((*Parser).term, ast.LESS_LESS, ast.GREATER_GREATER, ast.GREATER_GREATER_GREATER)
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

	if s.match(ast.RETURN, ast.BREAK, ast.CONTINUE) {
		j, err := s.jumpExpr()
		if err != nil {
			return nil, err
		}

		return j, nil
	}

	if s.match(ast.BANG, ast.MINUS, ast.TILDE) {
		operator := s.previous()
		right, err := s.unary()

		if err != nil {
			return nil, err
		}

		return ast.NewUnaryExpr(right, operator), nil
	}

	return s.call()
}

// jumpExpr parses return, break or continue in an expression (the keyword
// is consumed); return takes a value unless what follows ends the expression.
func (s *Parser) jumpExpr() (*ast.JumpExpr, error) {
	keyword := s.previous()

	switch keyword.TokenType {
	case ast.BREAK:
		return &ast.JumpExpr{Stmt: ast.NewBreakStmt(keyword)}, nil
	case ast.CONTINUE:
		return &ast.JumpExpr{Stmt: ast.NewContinueStmt(keyword)}, nil
	}

	var v ast.Expr

	switch s.peek().TokenType {
	case ast.SEMICOLON, ast.COMMA, ast.RIGHT_PAREN, ast.RIGHT_BRACKET, ast.RIGHT_BRACE, ast.COLON:
	default:
		var err error
		if v, err = s.assignment(); err != nil {
			return nil, err
		}
	}

	return &ast.JumpExpr{Stmt: ast.NewReturnStmt(v, keyword)}, nil
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

func (s *Parser) primary() (ast.Expr, error) {
	if s.match(ast.LEFT_BRACKET) {
		return s.arrayLiteral()
	}

	if s.match(ast.MATCH) {
		return s.matchExpr()
	}

	if s.match(ast.IF) {
		st, err := s.ifStatement()
		if err != nil {
			return nil, err
		}

		return st.AsExpr(), nil
	}

	if s.match(ast.TRY) {
		st, err := s.tryStatement()
		if err != nil {
			return nil, err
		}

		return st.AsExpr(), nil
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

		if s.qualifiedStructStart() {
			t, err := s.qualifiedName(prev)
			if err != nil {
				return nil, err
			}

			s.advance()

			return s.structLiteral(t)
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
		if name != "" && seen[name] {
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

// qualifiedStructStart reports whether Name, m.Name, Enum.Case or
// m.Enum.Case and then "{" start a struct literal; Name is the previous token.
func (s *Parser) qualifiedStructStart() bool {
	for at := 0; at <= 4; at += 2 {
		if s.isStructLiteralStart(at) {
			return true
		}

		if s.peekAt(at) != ast.DOT || s.peekAt(at+1) != ast.IDENTIFIER {
			return false
		}
	}

	return false
}

// qualifiedName parses the rest of Name, m.Name, Enum.Case or
// m.Enum.Case after its first name.
func (s *Parser) qualifiedName(first ast.Token) (*ast.VarExpr, error) {
	t := &ast.VarExpr{Name: first}
	if !s.match(ast.DOT) {
		return t, nil
	}

	name, err := s.consume(ast.IDENTIFIER, "Expect name after '.'.")
	if err != nil {
		return nil, err
	}

	t.Module, t.Name = &first, name
	if !s.match(ast.DOT) {
		return t, nil
	}

	variant, err := s.consume(ast.IDENTIFIER, "Expect variant name after '.'.")
	t.Case = &variant

	return t, err
}

// structLiteral parses the fields of a struct literal of type t; "{" has
// already been consumed.
func (s *Parser) structLiteral(t *ast.VarExpr) (ast.Expr, error) {
	pairs, err := s.structPairs()
	if err != nil {
		return nil, err
	}

	return &ast.StructExpr{Type: t, Pairs: pairs}, nil
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

// arrayLiteral parses [a, b, ...], or a map literal; the "[" has already
// been consumed.
func (s *Parser) arrayLiteral() (ast.Expr, error) {
	bracket := s.previous()

	if s.match(ast.COLON) {
		_, err := s.consume(ast.RIGHT_BRACKET, "Expect ']' after ':' of an empty map.")
		return &ast.MapExpr{Bracket: bracket}, err
	}

	var items []ast.Expr

	for !s.check(ast.RIGHT_BRACKET) {
		elem, err := s.item(true)
		if err != nil {
			return nil, err
		}

		if len(items) == 0 && !ast.IsSpread(elem) && s.check(ast.COLON) {
			return s.mapLiteral(bracket, elem)
		}

		items = append(items, elem)

		if !s.match(ast.COMMA) {
			break
		}
	}

	if _, err := s.consume(ast.RIGHT_BRACKET, "Expect ']' after array elements."); err != nil {
		return nil, err
	}

	return ast.NewArrayExpr(ast.ARRAY, items), nil
}

// mapLiteral parses the rest of [k: v, ...] after its first key.
func (s *Parser) mapLiteral(bracket ast.Token, key ast.Expr) (ast.Expr, error) {
	m := &ast.MapExpr{Bracket: bracket}
	seen := value.NewHashMap(0)

	for {
		keyTok := s.previous()

		if _, err := s.consume(ast.COLON, "Expect ':' after map key."); err != nil {
			return nil, err
		}

		if lit, ok := key.(*ast.LiteralExpr); ok {
			if _, dup, _ := seen.Get(lit.Value); dup {
				s.errors = append(s.errors, errors.NewSyntaxError(keyTok, "Duplicate map key."))
			}

			_ = seen.Set(lit.Value, value.True)
		}

		v, err := s.expressionWithStructs(structsAll)
		if err != nil {
			return nil, err
		}

		m.Items = append(m.Items, key, v)

		if !s.match(ast.COMMA) || s.check(ast.RIGHT_BRACKET) {
			break
		}

		if key, err = s.expressionWithStructs(structsAll); err != nil {
			return nil, err
		}
	}

	_, err := s.consume(ast.RIGHT_BRACKET, "Expect ']' after map entries.")

	return m, err
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
	if s.check(ast.LEFT_BRACE) {
		return s.functionBody()
	}

	e, err := s.assignment()

	return []ast.Stmt{ast.NewReturnStmt(e, arrow)}, err
}

func anonName(at ast.Token) ast.Token {
	name := ast.NewToken(ast.FUN, "", "", at.Line)
	name.Col = at.Col

	return name
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
