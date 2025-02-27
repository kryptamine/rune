package rune

import (
	"fmt"
	"strconv"

	"rune/pkg/ast"
	"rune/pkg/callable"
	"rune/pkg/errors"
	"slices"
)

type Parser struct {
	tokens  []ast.Token
	errors  []error
	current int
}

func ParseExpr(tokens []ast.Token) (ast.Expr, error) {
	parser := Parser{
		tokens:  tokens,
		current: 0,
	}

	return parser.expression()
}

func ParseStmts(tokens []ast.Token) ([]ast.Stmt, error) {
	var stmts []ast.Stmt

	parser := Parser{
		tokens:  tokens,
		current: 0,
	}

	for !parser.isAtEnd() {
		stmt, err := parser.declaration()

		if err != nil {
			return nil, err
		}

		stmts = append(stmts, stmt)
	}

	return stmts, nil
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

	if s.match(ast.LEFT_BRACE) {
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

func (s *Parser) forStatement() (ast.Stmt, error) {
	_, err := s.consume(ast.LEFT_PAREN, "Expect '(' after 'for'.")
	if err != nil {
		return nil, err
	}

	var initializer ast.Stmt

	if s.match(ast.SEMICOLON) {
		initializer = nil
	} else if s.match(ast.VAR) {
		initializer, err = s.varDeclaration()

		if err != nil {
			return nil, err
		}
	} else {
		initializer, err = s.expressionStatement()

		if err != nil {
			return nil, err
		}
	}

	var condition ast.Expr

	if !s.check(ast.SEMICOLON) {
		condition, err = s.expression()
		if err != nil {
			return nil, err
		}
	}

	_, err = s.consume(ast.SEMICOLON, "Expect ';' after condition.")
	if err != nil {
		return nil, err
	}

	var increment ast.Expr

	if !s.check(ast.RIGHT_PAREN) {
		increment, err = s.expression()

		if err != nil {
			return nil, err
		}
	}

	_, err = s.consume(ast.RIGHT_PAREN, "Expect ')' after clauses.")
	if err != nil {
		return nil, err
	}

	body, err := s.statement()
	if err != nil {
		return nil, err
	}

	if increment != nil {
		body = ast.NewBlockStmt([]ast.Stmt{body, ast.NewExprStmt(increment)})
	}

	if condition == nil {
		condition = ast.NewLiteralExpr(ast.TRUE, true)
	}

	body = ast.NewWhileStmt(condition, body)

	if initializer != nil {
		body = ast.NewBlockStmt([]ast.Stmt{initializer, body})
	}

	return body, nil
}

func (s *Parser) whileStatement() (ast.Stmt, error) {
	_, err := s.consume(ast.LEFT_PAREN, "Expect '(' after 'while'.")
	if err != nil {
		return nil, err
	}

	condition, err := s.expression()
	if err != nil {
		return nil, err
	}

	_, err = s.consume(ast.RIGHT_PAREN, "Expect ')' after condition.")
	if err != nil {
		return nil, err
	}

	body, err := s.statement()
	if err != nil {
		return nil, err
	}

	return ast.NewWhileStmt(condition, body), nil
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

func (s *Parser) block() ([]ast.Stmt, error) {
	var stmts []ast.Stmt

	for !s.check(ast.RIGHT_BRACE) && !s.isAtEnd() {
		stmt, err := s.declaration()

		if err != nil {
			return nil, err
		}

		stmts = append(stmts, stmt)
	}

	_, err := s.consume(ast.RIGHT_BRACE, "Expect '}' after block.")
	if err != nil {
		return nil, err
	}

	return stmts, nil
}

func (s *Parser) ifStatement() (ast.Stmt, error) {
	_, err := s.consume(ast.LEFT_PAREN, "Expect '(' after 'if'.")
	if err != nil {
		return nil, err
	}

	condition, err := s.expression()
	if err != nil {
		return nil, err
	}

	_, err = s.consume(ast.RIGHT_PAREN, "Expect ')' after condition.")
	if err != nil {
		return nil, err
	}

	then, err := s.statement()
	if err != nil {
		return nil, err
	}

	var el ast.Stmt

	if s.match(ast.ELSE) {
		el, err = s.statement()

		if err != nil {
			return nil, err
		}
	}

	return ast.NewIfStmt(condition, then, el), nil
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

func (s *Parser) declaration() (ast.Stmt, error) {
	if s.match(ast.FUN) {
		return s.function("function")
	}

	if s.match(ast.VAR) {
		return s.varDeclaration()
	}

	return s.statement()
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

	parameters := []ast.Token{}

	if !s.check(ast.RIGHT_PAREN) {
		for true {
			if len(parameters) >= callable.MaxArity {
				return nil, errors.NewRuntimeError(
					s.peek(),
					fmt.Sprintf("Error at '%s': Cannot have more than %d parameters.", s.peek().Lexeme, callable.MaxArity),
				)
			}

			param, err := s.consume(ast.IDENTIFIER, "Expect parameter name.")
			if err != nil {
				return nil, err
			}

			parameters = append(parameters, param)

			if !s.match(ast.COMMA) {
				break
			}
		}
	}

	_, err = s.consume(ast.RIGHT_PAREN, fmt.Sprintf(
		"Error at '%s': Expect ')' after parameters.",
		s.peek().Lexeme,
	))
	if err != nil {
		return nil, err
	}

	_, err = s.consume(ast.LEFT_BRACE, fmt.Sprintf(
		"Error at '%s': Expect '{' before function body.",
		s.peek().Lexeme,
	))
	if err != nil {
		return nil, err
	}

	body, err := s.block()
	if err != nil {
		return nil, err
	}

	return ast.NewFunctionStmt(name, parameters, body), nil
}

func (s *Parser) varDeclaration() (ast.Stmt, error) {
	name, err := s.consume(
		ast.IDENTIFIER,
		fmt.Sprintf("Error at '%s': Expect variable name.", s.peek().Lexeme),
	)

	if err != nil {
		return nil, err
	}

	var initializer ast.Expr

	if s.match(ast.EQUAL) {
		initializer, err = s.expression()

		if err != nil {
			return nil, err
		}
	}

	_, err = s.consume(ast.SEMICOLON, "Expect ';' after variable declaration.")
	if err != nil {
		return nil, err
	}

	return ast.NewVarStmt(initializer, name), nil
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

func (s *Parser) printStmt() (ast.Stmt, error) {
	expr, err := s.expression()

	if err != nil {
		return nil, err
	}

	_, err = s.consume(ast.SEMICOLON, fmt.Sprintf(
		"Error at '%s': Expect ';' after value.",
		s.peek().Lexeme,
	))
	if err != nil {
		return nil, err
	}

	return ast.NewPrintStmt(expr), nil
}

func (s *Parser) expression() (ast.Expr, error) {
	return s.assignment()
}

func (s *Parser) assignment() (ast.Expr, error) {
	expr, err := s.or()

	if err != nil {
		return nil, err
	}

	if s.match(ast.EQUAL) {
		value, err := s.assignment()

		if err != nil {
			return nil, err
		}

		if arrayExpr, ok := expr.(*ast.IndexExpr); ok {
			return ast.NewSetIndexExpr(arrayExpr.Token, arrayExpr.Array, arrayExpr.Index, value), nil
		}

		if s, ok := expr.(*ast.VarExpr); ok {
			token := s.Name

			return ast.NewAssignExpr(token, value), nil
		}

		return nil, errors.NewRuntimeError(s.peek(), "Error at '=': Invalid assignment target.")
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

func (s *Parser) unary() (ast.Expr, error) {
	for s.match(ast.BANG, ast.MINUS) {
		operator := s.previous()
		right, err := s.unary()

		if err != nil {
			return nil, err
		}

		return ast.NewUnaryExpr(right, operator), nil
	}

	return s.call()
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
		} else {
			break
		}
	}

	return expr, nil
}

func (s *Parser) finishCall(expr ast.Expr) (ast.Expr, error) {
	args := []ast.Expr{}

	if !s.check(ast.RIGHT_PAREN) {
		for true {
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

	for s.match(ast.SLASH, ast.STAR) {
		operator := s.previous()
		right, err := s.unary()

		if err != nil {
			return nil, err
		}

		expr = ast.NewBinaryExpr(expr, right, operator)
	}

	return expr, nil
}

func (s *Parser) primary() (ast.Expr, error) {
	// Parse object literal
	if s.match(ast.LEFT_BRACE) {
		pairs := make(map[string]ast.Expr)

		for !s.check(ast.RIGHT_BRACE) && !s.isAtEnd() {
			// Parse key (should be an identifier or string)
			key, err := s.consume(ast.IDENTIFIER, "Expect property name.")
			if err != nil {
				return nil, err
			}

			// Expect ':' after key
			_, err = s.consume(ast.COLON, "Expect ':' after property name.")
			if err != nil {
				return nil, err
			}

			value, err := s.expression()
			if err != nil {
				return nil, err
			}

			pairs[key.Lexeme] = value

			// Allow optional commas but not required before closing `}`
			if !s.match(ast.COMMA) {
				break
			}
		}

		// Expect closing `}`
		_, err := s.consume(ast.RIGHT_BRACE, "Expect '}' after object properties.")
		if err != nil {
			return nil, err
		}

		return ast.NewObjectExpr(ast.OBJECT, pairs), nil
	}

	// Parse array literal
	if s.match(ast.LEFT_BRACKET) {
		var items []ast.Expr

		if !s.check(ast.RIGHT_BRACKET) {
			for {
				elem, err := s.expression()
				if err != nil {
					return nil, err
				}
				items = append(items, elem)

				if !s.match(ast.COMMA) {
					break
				}
			}
		}

		_, err := s.consume(ast.RIGHT_BRACKET, "Expect ']' after array elements.")
		if err != nil {
			return nil, err
		}

		return ast.NewArrayExpr(ast.ARRAY, items), nil
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
		value, err := strconv.ParseFloat(prev.Literal, 64)

		if err != nil {
			return nil, errors.NewRuntimeError(prev, "Invalid number.")
		}

		return ast.NewLiteralExpr(ast.NUMBER, value), nil
	}

	if s.match(ast.STRING) {
		prev := s.previous()

		return ast.NewLiteralExpr(ast.STRING, prev.Literal), nil
	}

	if s.match(ast.IDENTIFIER) {
		prev := s.previous()

		return ast.NewVarExpr(prev), nil
	}

	if s.match(ast.LEFT_PAREN) {
		expr, err := s.expression()

		if err != nil {
			return nil, err
		}

		_, err = s.consume(ast.RIGHT_PAREN, fmt.Sprintf(
			"Error at '%s': Expect ')' after expression.",
			s.peek().Lexeme,
		))
		if err != nil {
			return nil, err
		}

		return ast.NewGroupingExpr(expr), nil
	}

	current := s.peek()

	return nil, errors.NewRuntimeError(
		current,
		fmt.Sprintf("Error at '%s': Expect expression.", current.Lexeme),
	)
}

func (s *Parser) match(tokenTypes ...ast.TokenType) bool {
	if slices.ContainsFunc(tokenTypes, s.check) {
		s.advance()
		return true
	}

	return false
}

func (s *Parser) check(tokenType ast.TokenType) bool {
	if s.isAtEnd() {
		return false
	}

	if s.peek().TokenType == tokenType {
		return true
	}

	return false
}

func (s *Parser) advance() ast.Token {
	if !s.isAtEnd() {
		s.current++
	}

	return s.previous()
}

func (s *Parser) consume(tokenType ast.TokenType, errMsg string) (ast.Token, error) {
	if s.check(tokenType) {
		return s.advance(), nil
	}

	return ast.Token{}, errors.NewRuntimeError(s.peek(), errMsg)
}

func (s *Parser) peek() ast.Token {
	return s.tokens[s.current]
}

func (s *Parser) isAtEnd() bool {
	return s.peek().TokenType == ast.EOF
}

func (s *Parser) previous() ast.Token {
	return s.tokens[s.current-1]
}
