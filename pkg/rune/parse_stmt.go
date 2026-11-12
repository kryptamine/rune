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

	if s.match(ast.STRUCT) {
		return s.structDeclaration()
	}

	if s.match(ast.IMPORT) {
		return nil, errors.NewSyntaxError(s.previous(), "Imports must come before other statements.")
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

		if s.match(ast.EQUAL) {
			at := s.peek()

			if def, err = s.expression(); err != nil {
				return "", field, err
			}

			if _, ok := literalOf(def); !ok {
				s.errors = append(s.errors, errors.NewSyntaxError(at, "Field default must be a literal."))
			}
		}

		fields = append(fields, ast.StructField{Name: field, Default: def})

		return field.Lexeme, field, nil
	})
	if err != nil {
		return nil, err
	}

	return ast.NewStructStmt(name, fields), nil
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
	parameters, patterns, err := s.parameters()
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

	return ast.NewFunctionStmt(name, parameters, append(patterns, body...)), nil
}

// parameters parses a parameter list up to and including ")", returning
// the destructuring of pattern parameters to run first in the body.
func (s *Parser) parameters() ([]ast.Binding, []ast.Stmt, error) {
	parameters := []ast.Binding{}

	var patterns []ast.Stmt

	if !s.check(ast.RIGHT_PAREN) {
		for {
			// Report but keep parsing: the parser is not in a confused state.
			if len(parameters) == maxArity {
				s.errors = append(s.errors, errors.NewSyntaxError(
					s.peek(),
					fmt.Sprintf("Cannot have more than %d parameters.", maxArity),
				))
			}

			param, pattern, err := s.parameter(len(parameters))
			if err != nil {
				return nil, nil, err
			}

			parameters = append(parameters, param)

			if pattern != nil {
				patterns = append(patterns, pattern)
			}

			if !s.match(ast.COMMA) {
				break
			}
		}
	}

	_, err := s.consume(ast.RIGHT_PAREN, "Expect ')' after parameters.")

	return parameters, patterns, err
}

// parameter parses a parameter: a binding, or a pattern, which binds a
// hidden parameter destructured at the start of the body.
func (s *Parser) parameter(i int) (ast.Binding, ast.Stmt, error) {
	if !s.match(ast.LEFT_BRACE) {
		param, err := s.binding("Expect parameter name.")
		return param, nil, err
	}

	fields, err := s.pattern(false)
	if err != nil {
		return ast.Binding{}, nil, err
	}

	hidden := s.previous()
	hidden.TokenType = ast.IDENTIFIER
	hidden.Lexeme = fmt.Sprintf("{%d}", i)

	return ast.Binding{Name: hidden}, ast.NewDestructureStmt(fields, ast.NewVarExpr(hidden)), nil
}

// pattern parses { a, mut b, c: d, "e": f }; the "{" has already been
// consumed. With mutable, every variable is mutable.
func (s *Parser) pattern(mutable bool) ([]ast.DestructureField, error) {
	var fields []ast.DestructureField

	for !s.check(ast.RIGHT_BRACE) && !s.isAtEnd() {
		f, err := s.patternField()
		if err != nil {
			return nil, err
		}

		f.Mutable = f.Mutable || mutable

		fields = append(fields, f)

		if !s.match(ast.COMMA) {
			break
		}
	}

	if len(fields) == 0 {
		s.errors = append(s.errors, errors.NewSyntaxError(s.peek(), "Expect property name in pattern."))
	}

	_, err := s.consume(ast.RIGHT_BRACE, "Expect '}' after pattern.")

	return fields, err
}

func (s *Parser) patternField() (ast.DestructureField, error) {
	if s.check(ast.MUT) {
		b, err := s.binding("Expect property name.")
		return ast.DestructureField{Key: b.Name, Binding: b}, err
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

		return ast.DestructureField{Key: key, Binding: ast.Binding{Name: key}}, nil
	}

	b, err := s.binding("Expect variable name.")

	return ast.DestructureField{Key: key, Binding: b}, err
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

func (s *Parser) varDeclaration() (ast.Stmt, error) {
	mutable := s.match(ast.MUT)

	if s.match(ast.LEFT_BRACE) {
		return s.destructureDeclaration(mutable)
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

// destructureDeclaration parses the rest of let { ... } = value; after
// "{"; with "let mut" every variable is mutable.
func (s *Parser) destructureDeclaration(mutable bool) (ast.Stmt, error) {
	fields, err := s.pattern(mutable)
	if err != nil {
		return nil, err
	}

	if _, err := s.consume(ast.EQUAL, "Expect '=' after pattern."); err != nil {
		return nil, err
	}

	initializer, err := s.expression()
	if err != nil {
		return nil, err
	}

	if _, err := s.consume(ast.SEMICOLON, "Expect ';' after variable declaration."); err != nil {
		return nil, err
	}

	return ast.NewDestructureStmt(fields, initializer), nil
}

func (s *Parser) statement() (ast.Stmt, error) {
	if s.match(ast.PRINT) {
		return s.printStmt()
	}

	if s.match(ast.RETURN) {
		return s.returnStmt()
	}

	if s.match(ast.WHILE) {
		return s.whileStatement()
	}

	if s.match(ast.TRY) {
		return s.tryStatement()
	}

	if s.match(ast.THROW) {
		return s.throwStatement()
	}

	if s.match(ast.BREAK, ast.CONTINUE) {
		return s.loopJumpStatement()
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

func (s *Parser) printStmt() (ast.Stmt, error) {
	expr, err := s.expression()

	if err != nil {
		return nil, err
	}

	_, err = s.consume(ast.SEMICOLON, "Expect ';' after value.")
	if err != nil {
		return nil, err
	}

	return ast.NewPrintStmt(expr), nil
}

func (s *Parser) returnStmt() (ast.Stmt, error) {
	keyword := s.previous()
	var value ast.Expr

	if !s.check(ast.SEMICOLON) {
		v, err := s.expression()

		value = v

		if err != nil {
			return nil, err
		}
	}

	_, err := s.consume(ast.SEMICOLON, "Expect ';' after return value.")
	if err != nil {
		return nil, err
	}

	return ast.NewReturnStmt(value, keyword), nil
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

// stmtKeyword reports whether t starts a statement that is no expression
// or declaration.
func stmtKeyword(t ast.TokenType) bool {
	switch t {
	case ast.PRINT, ast.RETURN, ast.THROW, ast.BREAK, ast.CONTINUE, ast.IF, ast.WHILE, ast.FOR, ast.TRY:
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

	if !s.isForInStart() {
		err := errors.NewSyntaxError(s.peek(), "Expect 'name in' after 'for'.")
		s.skipBadForHeader()

		return nil, err
	}

	return s.forInStatement(keyword)
}

// isForInStart reports whether the tokens after "for" are "name in" or
// "name, name in", each name optionally preceded by "mut".
func (s *Parser) isForInStart() bool {
	i := s.bindingEnd(0)

	if i >= 0 && s.peekAt(i) == ast.COMMA {
		i = s.bindingEnd(i + 1)
	}

	return i >= 0 && s.peekAt(i) == ast.IN
}

// forInStatement parses the rest of `for x in expr { ... }` or
// `for k, v in expr { ... }` after "for".
func (s *Parser) forInStatement(keyword ast.Token) (ast.Stmt, error) {
	// The bindings were checked by isForInStart, so they cannot fail.
	var key *ast.Binding

	val, _ := s.binding("")

	if s.match(ast.COMMA) {
		k := val
		key = &k
		val, _ = s.binding("")
	}

	s.advance() // "in", checked by isForInStart

	iterable, err := s.expressionWithStructs(structsNone)
	if err != nil {
		return nil, err
	}

	body, err := s.body("for-in clause")
	if err != nil {
		return nil, err
	}

	return ast.NewForInStmt(keyword, key, val, iterable, body), nil
}

// loopJumpStatement parses `break;` or `continue;` (the keyword is consumed).
func (s *Parser) loopJumpStatement() (ast.Stmt, error) {
	keyword := s.previous()

	if _, err := s.consume(ast.SEMICOLON, fmt.Sprintf("Expect ';' after '%s'.", keyword.Lexeme)); err != nil {
		return nil, err
	}

	if keyword.TokenType == ast.BREAK {
		return ast.NewBreakStmt(keyword), nil
	}

	return ast.NewContinueStmt(keyword), nil
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

	if s.match(ast.LEFT_PAREN) {
		name, err := s.consume(ast.IDENTIFIER, "Expect error variable name.")
		if err != nil {
			return c, err
		}

		c.Param = &name

		if s.match(ast.COLON) {
			for {
				t, err := s.consume(ast.IDENTIFIER, "Expect error type name.")
				if err != nil {
					return c, err
				}

				c.Types = append(c.Types, &ast.VarExpr{Name: t})

				if !s.match(ast.COMMA) {
					break
				}
			}
		}

		if _, err := s.consume(ast.RIGHT_PAREN, "Expect ')' after catch variable."); err != nil {
			return c, err
		}
	}

	if _, err := s.consume(ast.LEFT_BRACE, "Expect '{' after 'catch'."); err != nil {
		return c, err
	}

	handler, err := s.block()
	c.Body = handler

	return c, err
}

func (s *Parser) throwStatement() (ast.Stmt, error) {
	keyword := s.previous()

	value, err := s.expression()
	if err != nil {
		return nil, err
	}

	if _, err := s.consume(ast.SEMICOLON, "Expect ';' after thrown value."); err != nil {
		return nil, err
	}

	return ast.NewThrowStmt(value, keyword), nil
}
