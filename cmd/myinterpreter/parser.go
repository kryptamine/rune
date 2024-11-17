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

type ExprVisitor interface {
	visitBinaryExpr(binaryExpr *BinaryExpr) (any, error)
	visitLiteralExpr(literalExpr *LiteralExpr) (any, error)
	visitGroupingExpr(literalExpr *GroupingExpr) (any, error)
	visitUnaryExpr(UnaryExpr *UnaryExpr) (any, error)
	visitVarExpr(varExpr *VarExpr) (any, error)
}

type StmtVisitor interface {
	visitPrintStmt(printStmt *PrintStmt) error
	visitExprStmt(exprStmt *ExprStmt) error
	visitVarStmt(VarStmt *VarStmt) error
}

type VarStmt struct {
	initializer Node
	name        Token
}

type PrintStmt struct {
	expr Node
}

type ExprStmt struct {
	expr Node
}

type Stmt interface {
	accept(v StmtVisitor) error
}

type Node interface {
	accept(v ExprVisitor) (any, error)
}

type BinaryExpr struct {
	left     Node
	right    Node
	operator Token
}

type UnaryExpr struct {
	right    Node
	operator Token
}

type LiteralExpr struct {
	tokenType TokenType
	value     any
}

type VarExpr struct {
	name Token
}

type GroupingExpr struct {
	expr Node
}

func (n *PrintStmt) accept(v StmtVisitor) error {
	return v.visitPrintStmt(n)
}

func (n *ExprStmt) accept(v StmtVisitor) error {
	return v.visitExprStmt(n)
}

func (n *VarStmt) accept(v StmtVisitor) error {
	return v.visitVarStmt(n)
}

func (n *BinaryExpr) accept(v ExprVisitor) (any, error) {
	return v.visitBinaryExpr(n)
}

func (n *LiteralExpr) accept(v ExprVisitor) (any, error) {
	return v.visitLiteralExpr(n)
}

func (n *VarExpr) accept(v ExprVisitor) (any, error) {
	return v.visitVarExpr(n)
}

func (n *GroupingExpr) accept(v ExprVisitor) (any, error) {
	return v.visitGroupingExpr(n)
}

func (n *UnaryExpr) accept(v ExprVisitor) (any, error) {
	return v.visitUnaryExpr(n)
}

func Evaluate(tokens []Token) (Node, error) {
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

	return s.expressionStatement()
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

	var initializer Node

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

func (s *Parser) expression() (Node, error) {
	return s.equality()
}

func (s *Parser) equality() (Node, error) {
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

func (s *Parser) comparison() (Node, error) {
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

func (s *Parser) unary() (Node, error) {
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

func (s *Parser) term() (Node, error) {
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

func (s *Parser) factor() (Node, error) {
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

func (s *Parser) primary() (Node, error) {
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

	if s.match(IDENTIFIER) {
		prev := s.previous()

		return &VarExpr{
			name: prev,
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
