package main

import (
	"fmt"
)

type Parser struct {
	tokens  []Token
	errors  []error
	current int
}

type Visitor interface {
	visitBinaryExpr(binaryExpr *BinaryExpr)
	visitLiteralExpr(literalExpr *LiteralExpr)
	visitGroupingExpr(literalExpr *GroupingExpr)
	visitUnaryExpr(UnaryExpr *UnaryExpr)
}

type Node interface {
	accept(v Visitor)
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
	value string
}

type GroupingExpr struct {
	expr Node
}

func (n *BinaryExpr) accept(v Visitor) {
	v.visitBinaryExpr(n)
}

func (n *LiteralExpr) accept(v Visitor) {
	v.visitLiteralExpr(n)
}

func (n *GroupingExpr) accept(v Visitor) {
	v.visitGroupingExpr(n)
}

func (n *UnaryExpr) accept(v Visitor) {
	v.visitUnaryExpr(n)
}

func Parse(tokens []Token) Node {
	parser := Parser{
		tokens:  tokens,
		current: 0,
	}

	return parser.expression()
}

func (s *Parser) expression() Node {
	return s.term()
}

func (s *Parser) unary() Node {
	for s.match(BANG, MINUS) {
		operator := s.previous()
		right := s.unary()

		return &UnaryExpr{
			right:    right,
			operator: operator,
		}
	}

	return s.primary()
}

func (s *Parser) term() Node {
	expr := s.unary()

	for s.match(PLUS, MINUS) {
		operator := s.previous()
		right := s.unary()

		expr = &BinaryExpr{
			left:     expr,
			right:    right,
			operator: operator,
		}
	}

	return expr
}

func (s *Parser) primary() Node {
	if s.match(TRUE) {
		return &LiteralExpr{
			value: "true",
		}
	}

	if s.match(FALSE) {
		return &LiteralExpr{
			value: "false",
		}
	}

	if s.match(NIL) {
		return &LiteralExpr{
			value: "nil",
		}
	}

	if s.match(NUMBER, STRING) {
		prev := s.previous()

		return &LiteralExpr{
			value: prev.literal,
		}
	}

	if s.match(LEFT_PAREN) {
		expr := s.expression()

		s.consume(RIGHT_PAREN, fmt.Errorf("Expect ')' after expression."))

		return &GroupingExpr{
			expr: expr,
		}
	}

	return &LiteralExpr{}
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
