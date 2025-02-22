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

	if s.match(RETURN) {
		return s.returnStmt()
	}

	if s.match(WHILE) {
		return s.whileStatement()
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

	if s.match(FOR) {
		return s.forStatement()
	}

	if s.match(IF) {
		return s.ifStatement()
	}

	return s.expressionStatement()
}

func (s *Parser) forStatement() (Stmt, error) {
	_, err := s.consume(LEFT_PAREN, "Expect '(' after 'for'.")
	if err != nil {
		return nil, err
	}

	var initializer Stmt

	if s.match(SEMICOLON) {
		initializer = nil
	} else if s.match(VAR) {
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

	var condition Expr

	if !s.check(SEMICOLON) {
		condition, err = s.expression()
		if err != nil {
			return nil, err
		}
	}

	_, err = s.consume(SEMICOLON, "Expect ';' after condition.")
	if err != nil {
		return nil, err
	}

	var increment Expr

	if !s.check(RIGHT_PAREN) {
		increment, err = s.expression()

		if err != nil {
			return nil, err
		}
	}

	_, err = s.consume(RIGHT_PAREN, "Expect ')' after clauses.")
	if err != nil {
		return nil, err
	}

	body, err := s.statement()
	if err != nil {
		return nil, err
	}

	if increment != nil {
		body = &BlockStmt{
			stmts: []Stmt{
				body,
				&ExprStmt{
					expr: increment,
				},
			},
		}
	}

	if condition == nil {
		condition = &LiteralExpr{
			tokenType: TRUE,
			value:     true,
		}
	}

	body = &WhileStmt{
		condition: condition,
		body:      body,
	}

	if initializer != nil {
		body = &BlockStmt{
			stmts: []Stmt{
				initializer,
				body,
			},
		}
	}

	return body, nil
}

func (s *Parser) whileStatement() (Stmt, error) {
	_, err := s.consume(LEFT_PAREN, "Expect '(' after 'while'.")
	if err != nil {
		return nil, err
	}

	condition, err := s.expression()
	if err != nil {
		return nil, err
	}

	_, err = s.consume(RIGHT_PAREN, "Expect ')' after condition.")
	if err != nil {
		return nil, err
	}

	body, err := s.statement()
	if err != nil {
		return nil, err
	}

	return &WhileStmt{
		condition: condition,
		body:      body,
	}, nil
}

func (s *Parser) or() (Expr, error) {
	expr, err := s.and()
	if err != nil {
		return nil, err
	}

	for s.match(OR) {
		operator := s.previous()
		right, err := s.and()
		if err != nil {
			return nil, err
		}

		expr = &LogicalExpr{
			left:  expr,
			right: right,
			op:    operator,
		}
	}

	return expr, nil
}

func (s *Parser) and() (Expr, error) {
	expr, err := s.equality()
	if err != nil {
		return nil, err
	}

	for s.match(AND) {
		operator := s.previous()
		right, err := s.equality()
		if err != nil {
			return nil, err
		}

		expr = &LogicalExpr{
			left:  expr,
			right: right,
			op:    operator,
		}
	}

	return expr, nil
}

func (s *Parser) block() ([]Stmt, error) {
	var stmts []Stmt

	for !s.check(RIGHT_BRACE) && !s.isAtEnd() {
		stmt, err := s.declaration()

		if err != nil {
			return nil, err
		}

		stmts = append(stmts, stmt)
	}

	_, err := s.consume(RIGHT_BRACE, "Expect '}' after block.")
	if err != nil {
		return nil, err
	}

	return stmts, nil
}

func (s *Parser) ifStatement() (Stmt, error) {
	_, err := s.consume(LEFT_PAREN, "Expect '(' after 'if'.")
	if err != nil {
		return nil, err
	}

	condition, err := s.expression()
	if err != nil {
		return nil, err
	}

	_, err = s.consume(RIGHT_PAREN, "Expect ')' after condition.")
	if err != nil {
		return nil, err
	}

	then, err := s.statement()
	if err != nil {
		return nil, err
	}

	var el Stmt

	if s.match(ELSE) {
		el, err = s.statement()

		if err != nil {
			return nil, err
		}
	}

	return &IfStmt{
		condition: condition,
		then:      then,
		el:        el,
	}, nil
}

func (s *Parser) expressionStatement() (Stmt, error) {
	expr, err := s.expression()
	if err != nil {
		return nil, err
	}

	_, err = s.consume(SEMICOLON, "Expect ';' after expression.")
	if err != nil {
		return nil, err
	}

	return &ExprStmt{
		expr: expr,
	}, nil
}

func (s *Parser) declaration() (Stmt, error) {
	if s.match(FUN) {
		return s.function("function")
	}

	if s.match(VAR) {
		return s.varDeclaration()
	}

	return s.statement()
}

func (s *Parser) function(kind string) (Stmt, error) {
	name, err := s.consume(IDENTIFIER, fmt.Sprintf("Expect %s name.", kind))
	if err != nil {
		return nil, err
	}

	_, err = s.consume(LEFT_PAREN, fmt.Sprintf("Expect '(' after %s name.", kind))
	if err != nil {
		return nil, err
	}

	parameters := []Token{}

	if !s.check(RIGHT_PAREN) {
		for true {
			if len(parameters) >= 255 {
				return nil, NewRuntimeError(s.peek(), "Cannot have more than 255 parameters.")
			}

			param, err := s.consume(IDENTIFIER, "Expect parameter name.")
			if err != nil {
				return nil, err
			}

			parameters = append(parameters, param)

			if !s.match(COMMA) {
				break
			}
		}
	}

	_, err = s.consume(RIGHT_PAREN, fmt.Sprintf(
		"Error at '%s': Expect ')' after parameters.",
		s.peek().lexeme,
	))
	if err != nil {
		return nil, err
	}

	_, err = s.consume(LEFT_BRACE, fmt.Sprintf(
		"Error at '%s': Expect '{' before function body.",
		s.peek().lexeme,
	))
	if err != nil {
		return nil, err
	}

	body, err := s.block()
	if err != nil {
		return nil, err
	}

	return &FunctionStmt{
		name:       name,
		body:       body,
		parameters: parameters,
	}, nil
}

func (s *Parser) varDeclaration() (Stmt, error) {
	name, err := s.consume(IDENTIFIER, "Expect variable name.")
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

	_, err = s.consume(SEMICOLON, "Expect ';' after variable declaration.")
	if err != nil {
		return nil, err
	}

	return &VarStmt{
		initializer: initializer,
		name:        name,
	}, nil
}

func (s *Parser) returnStmt() (Stmt, error) {
	keyword := s.previous()
	var value Expr

	if !s.check(SEMICOLON) {
		v, err := s.expression()

		value = v

		if err != nil {
			return nil, err
		}
	}

	_, err := s.consume(SEMICOLON, "Expect ';' after return value.")
	if err != nil {
		return nil, err
	}

	return &ReturnStmt{
		value:   value,
		keyword: keyword,
	}, nil
}

func (s *Parser) printStmt() (Stmt, error) {
	expr, err := s.expression()

	if err != nil {
		return nil, err
	}

	_, err = s.consume(SEMICOLON, fmt.Sprintf(
		"Error at '%s': Expect ';' after value.",
		s.peek().lexeme,
	))
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

func (s *Parser) assignment() (Expr, error) {
	expr, err := s.or()

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

		return nil, NewRuntimeError(s.peek(), "Invalid assignment target.")
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

	return s.call()
}

func (s *Parser) call() (Expr, error) {
	expr, err := s.primary()
	if err != nil {
		return nil, err
	}

	for true {
		if s.match(LEFT_PAREN) {
			expr, err = s.finishCall(expr)

			if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}

	return expr, nil
}

func (s *Parser) finishCall(expr Expr) (Expr, error) {
	args := []Expr{}

	if !s.check(RIGHT_PAREN) {
		for true {
			arg, err := s.expression()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if !s.match(COMMA) {
				break
			}
		}
	}

	_, err := s.consume(RIGHT_PAREN, "Expect ')' after arguments.")
	if err != nil {
		return nil, err
	}

	return &CallExpr{
		token:  s.previous(),
		callee: expr,
		args:   args,
	}, nil
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

		_, err = s.consume(RIGHT_PAREN, fmt.Sprintf(
			"Error at '%s': Expect ')' after expression.",
			s.peek().lexeme,
		))
		if err != nil {
			return nil, err
		}

		return &GroupingExpr{
			expr: expr,
		}, nil
	}

	current := s.peek()

	return nil, NewRuntimeError(current, "Expect expression.")
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

func (s *Parser) consume(tokenType TokenType, errMsg string) (Token, error) {
	if s.check(tokenType) {
		return s.advance(), nil
	}

	return Token{}, NewRuntimeError(s.peek(), errMsg)
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
