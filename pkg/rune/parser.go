package rune

import (
	"rune/pkg/ast"
	"rune/pkg/errors"
	"slices"
	"strings"
)

type Parser struct {
	tokens  []ast.Token
	errors  []error
	current int
}

// ParseExpr parses a single expression and stops at the first error.
func ParseExpr(tokens []ast.Token) (ast.Expr, error) {
	parser := Parser{
		tokens:  tokens,
		current: 0,
	}

	return parser.expression()
}

// importDecl is `import a.b;`; tok is its first segment.
type importDecl struct {
	tok  ast.Token
	name string
}

// parseFile parses a file: its imports, then its statements, which in a
// module must be declarations. On a syntax error it records it, skips to
// the next statement boundary and keeps going, so all errors are reported
// at once.
func parseFile(tokens []ast.Token, module bool) ([]importDecl, []ast.Stmt, []error) {
	var imports []importDecl
	var stmts []ast.Stmt

	parser := Parser{
		tokens:  tokens,
		current: 0,
	}

	for parser.match(ast.IMPORT) {
		if imp, err := parser.importDecl(); err != nil {
			parser.recover(err)
		} else {
			imports = append(imports, imp)
		}
	}

	for !parser.isAtEnd() {
		tok := parser.peek()
		if stmt := parser.safeDeclaration(); stmt != nil {
			if module && !isDeclaration(stmt) {
				parser.errors = append(parser.errors, errors.NewSyntaxError(tok, "Only declarations can be at the top level of a module."))
			}

			stmts = append(stmts, stmt)
		}
	}

	if len(parser.errors) > 0 {
		return nil, nil, parser.errors
	}

	return imports, stmts, nil
}

func isDeclaration(stmt ast.Stmt) bool {
	switch stmt.(type) {
	case *ast.VarStmt, *ast.FunctionStmt, *ast.StructStmt:
		return true
	}

	return false
}

// importDecl parses a.b; after "import".
func (s *Parser) importDecl() (importDecl, error) {
	var name strings.Builder

	tok := s.peek()

	for {
		seg, err := s.consume(ast.IDENTIFIER, "Expect module name.")
		if err != nil {
			return importDecl{}, err
		}

		name.WriteString(seg.Lexeme)

		if !s.match(ast.DOT) {
			break
		}

		name.WriteByte('.')
	}

	if _, err := s.consume(ast.SEMICOLON, "Expect ';' after import."); err != nil {
		return importDecl{}, err
	}

	return importDecl{tok: tok, name: name.String()}, nil
}

// safeDeclaration parses a declaration and recovers from a syntax error in it.
// It returns nil if the declaration was invalid.
func (s *Parser) safeDeclaration() ast.Stmt {
	stmt, err := s.declaration()
	if err != nil {
		s.recover(err)
		return nil
	}

	return stmt
}

// recover records err and skips past the statement it is in.
func (s *Parser) recover(err error) {
	s.errors = append(s.errors, err)
	s.synchronize()
}

// synchronize discards tokens until it reaches a likely start of the next statement.
func (s *Parser) synchronize() {
	s.advance()

	for !s.isAtEnd() {
		if s.previous().TokenType == ast.SEMICOLON {
			return
		}

		switch s.peek().TokenType {
		case ast.FUN, ast.LET, ast.FOR, ast.IF, ast.WHILE, ast.PRINT, ast.RETURN, ast.TRY, ast.THROW,
			ast.BREAK, ast.CONTINUE, ast.STRUCT, ast.IMPORT:
			return
		}

		s.advance()
	}
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

	return ast.Token{}, errors.NewSyntaxError(s.peek(), errMsg)
}

// peekAt returns the type of the token n positions ahead of the current one,
// or EOF past the end.
func (s *Parser) peekAt(n int) ast.TokenType {
	if s.current+n >= len(s.tokens) {
		return ast.EOF
	}

	return s.tokens[s.current+n].TokenType
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
