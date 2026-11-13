package rune

import (
	"fmt"

	"rune/pkg/ast"
	"rune/pkg/errors"
)

func (s *Parser) declaration() (ast.Stmt, error) {
	// "fun (" starts an anonymous function expression, not a declaration.
	if s.check(ast.FUN) && s.peekAt(1) == ast.IDENTIFIER {
		s.advance()
		return s.function("function")
	}

	if s.match(ast.LET) {
		return s.varDeclaration()
	}

	if s.match(ast.CONST) {
		return s.constDeclaration()
	}

	if s.match(ast.STRUCT) {
		return s.structDeclaration()
	}

	if s.match(ast.ENUM) {
		return s.enumDeclaration()
	}

	if s.match(ast.IMPL) {
		return s.implDeclaration()
	}

	if s.match(ast.IMPORT) {
		return nil, errors.NewSyntaxError(s.previous(), "Imports must come before other statements.")
	}

	if s.match(ast.EXPORT) {
		return nil, errors.NewSyntaxError(s.previous(), "Only top-level declarations can be exported.")
	}

	return s.statement()
}

const maxFields = 255

// structDeclaration parses struct Name { a, b = 1 }; "struct" has already been consumed.
func (s *Parser) structDeclaration() (ast.Stmt, error) {
	name, err := s.consume(ast.IDENTIFIER, "Expect struct name.")
	if err != nil {
		return nil, err
	}

	if _, err := s.consume(ast.LEFT_BRACE, "Expect '{' after struct name."); err != nil {
		return nil, err
	}

	var fields []ast.StructField

	err = s.braceList("Duplicate field name.", "Expect '}' after struct fields.", func() (string, ast.Token, error) {
		field, err := s.consume(ast.IDENTIFIER, "Expect field name.")
		if err != nil {
			return "", field, err
		}

		if len(fields) == maxFields {
			s.errors = append(s.errors, errors.NewSyntaxError(field, fmt.Sprintf("Cannot have more than %d fields.", maxFields)))
		}

		var def ast.Expr
		var at ast.Token

		if s.match(ast.EQUAL) {
			at = s.peek()
			if def, err = s.expression(); err != nil {
				return "", field, err
			}
		}

		fields = append(fields, ast.StructField{Name: field, Default: def, DefaultAt: at})

		return field.Lexeme, field, nil
	})
	if err != nil {
		return nil, err
	}

	return ast.NewStructStmt(name, fields), nil
}

// enumDeclaration parses enum Name { A, B }; "enum" has already been consumed.
func (s *Parser) enumDeclaration() (ast.Stmt, error) {
	name, err := s.consume(ast.IDENTIFIER, "Expect enum name.")
	if err != nil {
		return nil, err
	}

	brace, err := s.consume(ast.LEFT_BRACE, "Expect '{' after enum name.")
	if err != nil {
		return nil, err
	}

	var variants []ast.Token

	err = s.braceList("Duplicate variant name.", "Expect '}' after enum variants.", func() (string, ast.Token, error) {
		v, err := s.consume(ast.IDENTIFIER, "Expect variant name.")
		if err != nil {
			return "", v, err
		}

		if v.Lexeme == enumValues || v.Lexeme == enumName {
			s.errors = append(s.errors, errors.NewSyntaxError(v, "A variant cannot be named '"+v.Lexeme+"'."))
		}

		variants = append(variants, v)

		return v.Lexeme, v, nil
	})
	if err != nil {
		return nil, err
	}

	if len(variants) == 0 {
		s.errors = append(s.errors, errors.NewSyntaxError(brace, "Expect at least one variant."))
	}

	return &ast.StructStmt{Name: name, Enum: true, Variants: variants}, nil
}

// implDeclaration parses impl Name { fun m(self) { ... } ... }; "impl" has
// already been consumed.
func (s *Parser) implDeclaration() (ast.Stmt, error) {
	name, err := s.consume(ast.IDENTIFIER, "Expect type name after 'impl'.")
	if err != nil {
		return nil, err
	}

	if _, err := s.consume(ast.LEFT_BRACE, "Expect '{' after type name."); err != nil {
		return nil, err
	}

	impl := &ast.ImplStmt{Type: name}

	for !s.check(ast.RIGHT_BRACE) && !s.isAtEnd() {
		if _, err := s.consume(ast.FUN, "Expect 'fun' in impl block."); err != nil {
			return nil, err
		}

		fn, err := s.function("method")
		if err != nil {
			return nil, err
		}

		impl.Methods = append(impl.Methods, ast.ImplMethod{Fn: fn.(*ast.FunctionStmt)})
	}

	if _, err := s.consume(ast.RIGHT_BRACE, "Expect '}' after methods."); err != nil {
		return nil, err
	}

	return impl, nil
}

func (s *Parser) function(kind string) (ast.Stmt, error) {
	name, err := s.consume(ast.IDENTIFIER, fmt.Sprintf("Expect %s name.", kind))
	if err != nil {
		return nil, err
	}

	_, err = s.consume(ast.LEFT_PAREN, fmt.Sprintf("Expect '(' after %s name.", kind))
	if err != nil {
		return nil, err
	}

	fn, err := s.functionRest(name)
	if err != nil {
		return nil, err
	}

	return fn, nil
}

const maxArity = 8

// functionRest parses a function's parameter list and body; the opening
// "(" has already been consumed.
func (s *Parser) functionRest(name ast.Token) (*ast.FunctionStmt, error) {
	params, err := s.parameters()
	if err != nil {
		return nil, err
	}

	_, err = s.consume(ast.LEFT_BRACE, "Expect '{' before function body.")
	if err != nil {
		return nil, err
	}

	body, err := s.block()
	if err != nil {
		return nil, err
	}

	return params.function(name, body), nil
}

// params is a parameter list; patterns destructure pattern parameters
// first in the body.
type params struct {
	list     []ast.Binding
	defaults []ast.Expr
	patterns []ast.Stmt
	rest     bool
}

func (ps params) function(name ast.Token, body []ast.Stmt) *ast.FunctionStmt {
	fn := ast.NewFunctionStmt(name, ps.list, append(ps.patterns, body...))
	fn.Rest = ps.rest
	fn.Defaults = ps.defaults

	return fn
}

// parameters parses a parameter list up to and including ")".
func (s *Parser) parameters() (params, error) {
	ps := params{list: []ast.Binding{}}
	sawDefault := false

	for !s.check(ast.RIGHT_PAREN) && !ps.rest {
		// Report but keep parsing: the parser is not in a confused state.
		if len(ps.list) == maxArity {
			s.errors = append(s.errors, errors.NewSyntaxError(
				s.peek(),
				fmt.Sprintf("Cannot have more than %d parameters.", maxArity),
			))
		}

		ps.rest = s.match(ast.DOT_DOT_DOT)

		param, pattern, err := s.parameter(len(ps.list), ps.rest)
		if err != nil {
			return ps, err
		}

		ps.list = append(ps.list, param)

		def, err := s.paramDefault(ps.rest)
		if err != nil {
			return ps, err
		}

		if def == nil && sawDefault && !ps.rest {
			s.errors = append(s.errors, errors.NewSyntaxError(param.Name, "A parameter without a default cannot follow one with a default."))
		}

		sawDefault = sawDefault || def != nil
		ps.defaults = append(ps.defaults, def)

		if pattern != nil {
			ps.patterns = append(ps.patterns, pattern)
		}

		if !s.match(ast.COMMA) {
			break
		}
	}

	if ps.rest && !s.check(ast.RIGHT_PAREN) {
		return ps, errors.NewSyntaxError(s.peek(), "Rest parameter must be last.")
	}

	_, err := s.consume(ast.RIGHT_PAREN, "Expect ')' after parameters.")

	return ps, err
}

// paramDefault parses the "= value" of the parameter just parsed, if any.
func (s *Parser) paramDefault(rest bool) (ast.Expr, error) {
	if !s.match(ast.EQUAL) {
		return nil, nil
	}

	if rest {
		return nil, errors.NewSyntaxError(s.previous(), "A rest parameter cannot have a default.")
	}

	return s.expressionWithStructs(structsAll)
}

// parameter parses a parameter: a binding, or a pattern, which binds a
// hidden parameter destructured at the start of the body. A rest
// parameter is a binding.
func (s *Parser) parameter(i int, rest bool) (ast.Binding, ast.Stmt, error) {
	if rest {
		param, err := s.binding("Expect parameter name.")
		return param, nil, err
	}

	return s.bindingOrPattern(fmt.Sprintf("{%d}", i), "Expect parameter name.")
}

func (s *Parser) varDeclaration() (ast.Stmt, error) {
	mutable := s.match(ast.MUT)

	if s.match(ast.LEFT_BRACE) {
		d, err := s.pattern(mutable)
		if err != nil {
			return nil, err
		}

		d.Initializer, err = s.patternInitializer()

		return d, err
	}

	if s.match(ast.LEFT_BRACKET) {
		d, err := s.arrayPattern(mutable)
		if err != nil {
			return nil, err
		}

		d.Initializer, err = s.patternInitializer()

		return d, err
	}

	name, err := s.consume(ast.IDENTIFIER, "Expect variable name.")
	if err != nil {
		return nil, err
	}

	binding := ast.Binding{Name: name, Mutable: mutable}

	var initializer ast.Expr

	if s.match(ast.EQUAL) {
		initializer, err = s.expression()

		if err != nil {
			return nil, err
		}
	} else if !binding.Mutable {
		s.errors = append(s.errors, errors.NewSyntaxError(binding.Name, "Immutable variable must be initialized."))
	}

	_, err = s.consume(ast.SEMICOLON, "Expect ';' after variable declaration.")
	if err != nil {
		return nil, err
	}

	return ast.NewVarStmt(initializer, binding), nil
}

// constDeclaration parses const NAME = value; "const" has already been consumed.
func (s *Parser) constDeclaration() (ast.Stmt, error) {
	name, err := s.consume(ast.IDENTIFIER, "Expect constant name.")
	if err != nil {
		return nil, err
	}

	binding := ast.Binding{Name: name}
	if binding.Discards() {
		s.errors = append(s.errors, errors.NewSyntaxError(name, "Expect constant name."))
	}

	if _, err := s.consume(ast.EQUAL, "Expect '=' after constant name."); err != nil {
		return nil, err
	}

	initializer, err := s.expression()
	if err != nil {
		return nil, err
	}

	if _, err := s.consume(ast.SEMICOLON, "Expect ';' after constant declaration."); err != nil {
		return nil, err
	}

	return &ast.VarStmt{Binding: binding, Initializer: initializer, Const: true}, nil
}

func (s *Parser) statement() (ast.Stmt, error) {
	if s.match(ast.RETURN, ast.BREAK, ast.CONTINUE) {
		return s.jumpStatement()
	}

	if s.match(ast.WHILE) {
		return s.whileStatement()
	}

	if s.match(ast.TRY) {
		return s.tryStatement()
	}

	if s.match(ast.MATCH) {
		m, err := s.matchExpr()
		if err != nil {
			return nil, err
		}

		s.match(ast.SEMICOLON)

		return &ast.MatchStmt{Match: m}, nil
	}

	// "{ key: ..." can only be an object literal; anything else after "{" is a block.
	if !s.isObjectLiteralStart() && s.match(ast.LEFT_BRACE) {
		block, err := s.block()
		if err != nil {
			return nil, err
		}

		return ast.NewBlockStmt(block), nil
	}

	if s.match(ast.FOR) {
		return s.forStatement()
	}

	if s.match(ast.IF) {
		return s.ifStatement()
	}

	return s.expressionStatement()
}

// jumpStatement parses `return x;`, `break;` or `continue;` (the keyword
// is consumed).
func (s *Parser) jumpStatement() (ast.Stmt, error) {
	keyword := s.previous()

	j, err := s.jumpExpr()
	if err != nil {
		return nil, err
	}

	if _, err := s.consume(ast.SEMICOLON, fmt.Sprintf("Expect ';' after '%s'.", keyword.Lexeme)); err != nil {
		return nil, err
	}

	return j.Stmt, nil
}

func (s *Parser) expressionStatement() (ast.Stmt, error) {
	expr, err := s.expression()
	if err != nil {
		return nil, err
	}

	_, err = s.consume(ast.SEMICOLON, "Expect ';' after expression.")
	if err != nil {
		return nil, err
	}

	return ast.NewExprStmt(expr), nil
}

// braceBody parses a block unless "{" starts an object literal; ok
// reports whether it did.
func (s *Parser) braceBody() (stmts []ast.Stmt, ok bool, err error) {
	if s.isObjectLiteralStart() || !s.match(ast.LEFT_BRACE) {
		return nil, false, nil
	}

	defer s.allowStructs(structsAll)()

	stmts, err = s.block()

	return stmts, true, err
}

func (s *Parser) block() ([]ast.Stmt, error) {
	var stmts []ast.Stmt

	outer := s.blockDepth
	s.blockDepth = s.depth[s.current]
	defer func() { s.blockDepth = outer }()

	for !s.check(ast.RIGHT_BRACE) && !s.isAtEnd() {
		if stmt := s.safeDeclaration(); stmt != nil {
			stmts = append(stmts, stmt)
		}
	}

	_, err := s.consume(ast.RIGHT_BRACE, "Expect '}' after block.")
	if err != nil {
		return nil, err
	}

	return stmts, nil
}

func (s *Parser) ifStatement() (ast.Stmt, error) {
	condition, err := s.expressionWithStructs(structsNone)
	if err != nil {
		return nil, err
	}

	then, err := s.body("if condition")
	if err != nil {
		return nil, err
	}

	var el ast.Stmt

	if s.match(ast.ELSE) {
		if s.match(ast.IF) {
			el, err = s.ifStatement()
		} else {
			el, err = s.body("'else'")
		}

		if err != nil {
			return nil, err
		}
	}

	return ast.NewIfStmt(condition, then, el), nil
}

// stmtKeyword reports whether t starts a statement that is no
// declaration: a block statement or a jump.
func stmtKeyword(t ast.TokenType) bool {
	switch t {
	case ast.RETURN, ast.THROW, ast.BREAK, ast.CONTINUE:
		return true
	}

	return blockKeyword(t)
}

// blockKeyword reports whether t starts a statement that is no expression.
func blockKeyword(t ast.TokenType) bool {
	switch t {
	case ast.IF, ast.WHILE, ast.FOR, ast.TRY:
		return true
	}

	return false
}

// body parses an if/else/while/for body: a block, or a bare statement
// starting with a token that cannot continue the paren-free header.
func (s *Parser) body(after string) (ast.Stmt, error) {
	if t := s.peek().TokenType; t == ast.LEFT_BRACE || t == ast.IDENTIFIER || t == ast.MATCH || stmtKeyword(t) {
		return s.statement()
	}

	return nil, errors.NewSyntaxError(s.peek(), fmt.Sprintf("Expect '{' after %s.", after))
}

// skipBadForHeader moves past a bad for header and its body, if it has one,
// without leaving the enclosing block, so one bad header reports one error.
func (s *Parser) skipBadForHeader() {
	for depth := 0; !s.isAtEnd(); s.advance() {
		switch s.peek().TokenType {
		case ast.LEFT_PAREN, ast.LEFT_BRACKET:
			depth++
		case ast.RIGHT_PAREN, ast.RIGHT_BRACKET:
			depth--
		case ast.LEFT_BRACE:
			if depth <= 0 {
				s.skipBraces()
				return
			}
		}

		if depth <= 0 && s.peekAt(1) == ast.RIGHT_BRACE {
			return
		}
	}
}

// skipBraces moves from a '{' to its matching '}'.
func (s *Parser) skipBraces() {
	for depth := 0; !s.isAtEnd(); s.advance() {
		switch s.peek().TokenType {
		case ast.LEFT_BRACE:
			depth++
		case ast.RIGHT_BRACE:
			if depth--; depth == 0 {
				return
			}
		}
	}
}

func (s *Parser) whileStatement() (ast.Stmt, error) {
	condition, err := s.expressionWithStructs(structsNone)
	if err != nil {
		return nil, err
	}

	body, err := s.body("while condition")
	if err != nil {
		return nil, err
	}

	return ast.NewWhileStmt(condition, body), nil
}

func (s *Parser) forStatement() (ast.Stmt, error) {
	keyword := s.previous()
	start, nerrs := s.current, len(s.errors)

	key, val, unpack, ok := s.forBindings()
	if !ok {
		s.current, s.errors = start, s.errors[:nerrs]
		err := errors.NewSyntaxError(s.peek(), "Expect 'name in' after 'for'.")
		s.skipBadForHeader()

		return nil, err
	}

	iterable, err := s.expressionWithStructs(structsNone)
	if err != nil {
		return nil, err
	}

	body, err := s.body("for-in clause")
	if err != nil {
		return nil, err
	}

	if unpack != nil {
		if b, ok := body.(*ast.BlockStmt); ok {
			b.Stmts = append([]ast.Stmt{unpack}, b.Stmts...)
		} else {
			body = ast.NewBlockStmt([]ast.Stmt{unpack, body})
		}
	}

	return ast.NewForInStmt(keyword, key, val, iterable, body), nil
}

// forBindings parses "x in" or "k, x in" after "for": names, and x may be
// a pattern, which unpack destructures; ok is false if they are not that.
func (s *Parser) forBindings() (key *ast.Binding, val ast.Binding, unpack ast.Stmt, ok bool) {
	val, unpack, err := s.bindingOrPattern("{for}", "")
	if err != nil {
		return nil, val, nil, false
	}

	if s.match(ast.COMMA) {
		comma, k, keyPattern := s.previous(), val, unpack
		key = &k

		if val, unpack, err = s.bindingOrPattern("{for}", ""); err != nil {
			return nil, val, nil, false
		}

		if keyPattern != nil {
			s.errors = append(s.errors, errors.NewSyntaxError(comma, "A for-in key cannot be a pattern."))
		}
	}

	return key, val, unpack, s.match(ast.IN)
}

// tryStatement parses `try { ... }` and its catch clauses: `catch { ... }`,
// `catch (name) { ... }` or `catch (name: Type, ...) { ... }`.
func (s *Parser) tryStatement() (ast.Stmt, error) {
	if _, err := s.consume(ast.LEFT_BRACE, "Expect '{' after 'try'."); err != nil {
		return nil, err
	}

	body, err := s.block()
	if err != nil {
		return nil, err
	}

	if _, err := s.consume(ast.CATCH, "Expect 'catch' after try block."); err != nil {
		return nil, err
	}

	var catches []ast.CatchClause

	for {
		c, err := s.catchClause()
		if err != nil {
			return nil, err
		}

		catches = append(catches, c)

		if !s.match(ast.CATCH) {
			break
		}
	}

	return ast.NewTryStmt(ast.NewBlockStmt(body), catches), nil
}

func (s *Parser) catchClause() (ast.CatchClause, error) {
	c := ast.CatchClause{Keyword: s.previous()}

	if s.match(ast.IDENTIFIER) {
		name := s.previous()
		c.Param = &name

		if s.match(ast.IS) {
			types, err := s.typeList()
			if err != nil {
				return c, err
			}

			c.Types = types
		}
	}

	if _, err := s.consume(ast.LEFT_BRACE, "Expect '{' after 'catch'."); err != nil {
		return c, err
	}

	handler, err := s.block()
	c.Body = handler

	return c, err
}

// typeList parses the types after 'is': T, m.T, nil, struct or enum, joined by '|'.
func (s *Parser) typeList() ([]*ast.VarExpr, error) {
	var types []*ast.VarExpr

	for {
		typ, err := s.typeName()
		if err != nil {
			return nil, err
		}

		types = append(types, typ)

		if next := s.peekAt(1); !s.check(ast.PIPE) || next != ast.IDENTIFIER && !isTypeKeyword(next) {
			return types, nil
		}

		s.advance()
	}
}

// isTypeKeyword reports whether t is a keyword that names a type after 'is'.
func isTypeKeyword(t ast.TokenType) bool { return t == ast.NIL || t == ast.STRUCT || t == ast.ENUM }

func (s *Parser) typeName() (*ast.VarExpr, error) {
	if isTypeKeyword(s.peek().TokenType) {
		return &ast.VarExpr{Name: s.advance()}, nil
	}

	t, err := s.consume(ast.IDENTIFIER, "Expect type name after 'is'.")
	if err != nil {
		return nil, err
	}

	typ := &ast.VarExpr{Name: t}
	if s.match(ast.DOT) {
		if typ.Name, err = s.consume(ast.IDENTIFIER, "Expect type name after '.'."); err != nil {
			return nil, err
		}

		typ.Module = &t
	}

	return typ, nil
}
