package main

import (
	"fmt"
	"strconv"
)

type Parser struct {
	tokens  []Token
	errors  []error
	current int
}

func Evaluate(tokens []Token) (Expr, error) {
	parser := Parser{
		tokens:  tokens,
		current: 0,
	}

	return parser.expression()
}

func Parse(tokens []Token) ([]Stmt, error) {
	var stmts []Stmt

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

func (s *Parser) statement() (Stmt, error) {
	if s.match(PRINT) {
		return s.printStmt()
	}

	if s.match(LEFT_BRACE) {
		block, err := s.block()
		if err != nil {
			return nil, err
		}

		return &BlockStmt{
			stmts: block,
		}, nil
	}

	return s.expressionStatement()
}

func (s *Parser) block() ([]Stmt, error) {
	var stmts []Stmt

	for !s.match(RIGHT_BRACE) && !s.isAtEnd() {
		stmt, err := s.declaration()

		if err != nil {
			return nil, err
		}

		stmts = append(stmts, stmt)
	}

	_, err := s.consume(RIGHT_BRACE, fmt.Errorf("Expect '}' after block."))
	if err != nil {
		return nil, err
	}

	return stmts, nil
}

func (s *Parser) expressionStatement() (Stmt, error) {
	expr, err := s.expression()
	if err != nil {
		return nil, err
	}

	_, err = s.consume(SEMICOLON, fmt.Errorf("Expect ';' after expression."))
	if err != nil {
		return nil, err
	}

	return &ExprStmt{
		expr: expr,
	}, nil
}

func (s *Parser) declaration() (Stmt, error) {
	if s.match(VAR) {
		return s.varDeclaration()
	}

	return s.statement()
}

func (s *Parser) varDeclaration() (Stmt, error) {
	name, err := s.consume(IDENTIFIER, fmt.Errorf("Expect variable name."))
	if err != nil {
		return nil, err
	}

	var initializer Expr

	if s.match(EQUAL) {
		initializer, err = s.expression()

		if err != nil {
			return nil, err
		}
	}

	_, err = s.consume(SEMICOLON, fmt.Errorf("Expect ';' after variable declaration."))
	if err != nil {
		return nil, err
	}

	return &VarStmt{
		initializer: initializer,
		name:        name,
	}, nil
}

func (s *Parser) printStmt() (Stmt, error) {
	expr, err := s.expression()

	if err != nil {
		return nil, err
	}

	_, err = s.consume(SEMICOLON, fmt.Errorf("Expect ';' after value."))

	if err != nil {
		return nil, err
	}

	return &PrintStmt{
		expr: expr,
	}, nil
}

func (s *Parser) expression() (Expr, error) {
	return s.assignment()
}

func (s *Parser) printTokens() {
	fmt.Println(fmt.Sprintf("current: %d", s.current))

	for i := s.current; i < len(s.tokens); i++ {
		fmt.Println(s.tokens[i])
	}
}

func (s *Parser) assignment() (Expr, error) {
	expr, err := s.equality()

	if err != nil {
		return nil, err
	}

	if s.match(EQUAL) {
		value, err := s.assignment()

		if err != nil {
			return nil, err
		}

		if s, ok := expr.(*VarExpr); ok {
			token := s.name

			return &AssignExpr{
				name:  token,
				value: value,
			}, nil
		}

		return nil, fmt.Errorf("Invalid assignment target.")
	}

	return expr, nil
}

func (s *Parser) equality() (Expr, error) {
	expr, err := s.comparison()

	if err != nil {
		return nil, err
	}

	for s.match(BANG_EQUAL, EQUAL_EQUAL) {
		operator := s.previous()
		right, err := s.comparison()

		if err != nil {
			return nil, err
		}

		expr = &BinaryExpr{
			left:     expr,
			right:    right,
			operator: operator,
		}
	}

	return expr, nil
}

func (s *Parser) comparison() (Expr, error) {
	expr, err := s.term()

	if err != nil {
		return nil, err
	}

	for s.match(GREATER, GREATER_EQUAL, LESS, LESS_EQUAL) {
		operator := s.previous()
		right, err := s.term()

		if err != nil {
			return nil, err
		}

		expr = &BinaryExpr{
			left:     expr,
			right:    right,
			operator: operator,
		}
	}

	return expr, nil
}

func (s *Parser) unary() (Expr, error) {
	for s.match(BANG, MINUS) {
		operator := s.previous()
		right, err := s.unary()

		if err != nil {
			return nil, err
		}

		return &UnaryExpr{
			right:    right,
			operator: operator,
		}, nil
	}

	return s.primary()
}

func (s *Parser) term() (Expr, error) {
	expr, err := s.factor()

	if err != nil {
		return nil, err
	}

	for s.match(PLUS, MINUS) {
		operator := s.previous()
		right, err := s.factor()
		if err != nil {
			return nil, err
		}

		expr = &BinaryExpr{
			left:     expr,
			right:    right,
			operator: operator,
		}
	}

	return expr, nil
}

func (s *Parser) factor() (Expr, error) {
	expr, err := s.unary()

	if err != nil {
		return nil, err
	}

	for s.match(SLASH, STAR) {
		operator := s.previous()
		right, err := s.unary()

		if err != nil {
			return nil, err
		}

		expr = &BinaryExpr{
			left:     expr,
			right:    right,
			operator: operator,
		}
	}

	return expr, nil
}

func (s *Parser) primary() (Expr, error) {
	if s.match(TRUE) {
		return &LiteralExpr{
			value:     true,
			tokenType: TRUE,
		}, nil
	}

	if s.match(FALSE) {
		return &LiteralExpr{
			value:     false,
			tokenType: FALSE,
		}, nil
	}

	if s.match(NIL) {
		return &LiteralExpr{
			value:     nil,
			tokenType: NIL,
		}, nil
	}

	if s.match(NUMBER) {
		prev := s.previous()
		value, _ := strconv.ParseFloat(prev.literal, 64)

		return &LiteralExpr{
			value:     value,
			tokenType: NUMBER,
		}, nil
	}

	if s.match(STRING) {
		prev := s.previous()

		return &LiteralExpr{
			value:     prev.literal,
			tokenType: STRING,
		}, nil
	}

	if s.match(IDENTIFIER) {
		prev := s.previous()

		return &VarExpr{
			name: prev,
		}, nil
	}

	if s.match(LEFT_PAREN) {
		expr, err := s.expression()

		if err != nil {
			return nil, err
		}

		_, err = s.consume(RIGHT_PAREN, fmt.Errorf("Expect ')' after expression."))
		if err != nil {
			return nil, err
		}

		return &GroupingExpr{
			expr: expr,
		}, nil
	}

	return nil, fmt.Errorf("Expect expression at: %s", s.peek())
}

func (s *Parser) match(tokenTypes ...TokenType) bool {
	for _, tokenType := range tokenTypes {
		if s.check(tokenType) {
			s.advance()

			return true
		}

	}

	return false
}

func (s *Parser) check(tokenType TokenType) bool {
	if s.isAtEnd() {
		return false
	}

	if s.peek().tokenType == tokenType {
		return true
	}

	return false
}

func (s *Parser) advance() Token {
	if !s.isAtEnd() {
		s.current++
	}

	return s.previous()
}

func (s *Parser) consume(tokenType TokenType, err error) (Token, error) {
	if s.check(tokenType) {
		return s.advance(), nil
	}

	return Token{}, err
}

func (s *Parser) peek() Token {
	return s.tokens[s.current]
}

func (s *Parser) isAtEnd() bool {
	return s.peek().tokenType == EOF
}

func (s *Parser) previous() Token {
	return s.tokens[s.current-1]
}
