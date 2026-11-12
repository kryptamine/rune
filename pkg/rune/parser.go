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
	structs structMode
	// depth[i] is the number of braces open before tokens[i], a closing one
	// included; blockDepth that of the statements of the block being parsed,
	// 0 at the top level. They tell synchronize where a statement ends.
	depth      []int
	blockDepth int
}

func newParser(tokens []ast.Token) *Parser {
	return &Parser{tokens: tokens, depth: braceDepths(tokens)}
}

// structMode says which "Name {" start a struct literal.
type structMode uint8

const (
	structsAll structMode = iota
	// structsNonEmpty is for a match head, where "Name { }" is an empty body.
	structsNonEmpty
	// structsNone is for an if/while/for header, where "Name {" opens the body.
	structsNone
)

// ParseExpr parses a single expression and stops at the first error.
func ParseExpr(tokens []ast.Token) (ast.Expr, error) {
	return newParser(tokens).expression()
}

// importDecl is `import a.b;` or `import a.b as m;`; tok is its first
// segment, as the name it binds: its last segment or alias. A name with
// leading dots is relative: `.a` is next to the importing file, `..a` in
// its parent directory. The loader sets file to the imported file's index.
type importDecl struct {
	tok  ast.Token
	name string
	dots int
	as   ast.Token
	file int
}

// fileAST is a parsed file; exports are its exported declarations.
type fileAST struct {
	file    int
	path    string
	imports []importDecl
	stmts   []ast.Stmt
	exports []ast.Stmt
}

// parseFile parses a file: its imports, then its statements, which in a
// module must be declarations. On a syntax error it records it, skips to
// the next statement boundary and keeps going, so all errors are reported
// at once.
func parseFile(tokens []ast.Token, module bool) (*fileAST, []error) {
	file := &fileAST{}

	parser := newParser(tokens)

	for parser.match(ast.IMPORT) {
		if imp, err := parser.importDecl(); err != nil {
			parser.recover(err)
		} else {
			file.imports = append(file.imports, imp)
		}
	}

	for !parser.isAtEnd() {
		exported := parser.match(ast.EXPORT)
		tok := parser.peek()

		stmt := parser.safeDeclaration()
		if stmt == nil {
			continue
		}

		_, isImpl := stmt.(*ast.ImplStmt)

		switch {
		case exported && isImpl:
			parser.errors = append(parser.errors, errors.NewSyntaxError(tok, "Cannot export an impl block; methods go wherever their type does."))
		case isDeclaration(stmt):
		case exported:
			parser.errors = append(parser.errors, errors.NewSyntaxError(tok, "Expect declaration after 'export'."))
		case module:
			parser.errors = append(parser.errors, errors.NewSyntaxError(tok, "Only declarations can be at the top level of a module."))
		}

		file.stmts = append(file.stmts, stmt)

		if exported {
			file.exports = append(file.exports, stmt)
		}
	}

	if len(parser.errors) > 0 {
		return nil, parser.errors
	}

	return file, nil
}

func isDeclaration(stmt ast.Stmt) bool {
	switch stmt.(type) {
	case *ast.StructStmt, *ast.ImplStmt:
		return true
	}

	return len(ast.Declared(stmt)) > 0
}

// importDecl parses a.b [as m]; after "import".
func (s *Parser) importDecl() (importDecl, error) {
	var name strings.Builder

	tok := s.peek()
	dots := 0

	for s.match(ast.DOT, ast.DOT_DOT, ast.DOT_DOT_DOT) {
		dots += len(s.previous().Lexeme)
	}

	name.WriteString(strings.Repeat(".", dots))

	var as ast.Token

	for {
		seg, err := s.consume(ast.IDENTIFIER, "Expect module name.")
		if err != nil {
			return importDecl{}, err
		}

		name.WriteString(seg.Lexeme)
		as = seg

		if !s.match(ast.DOT) {
			break
		}

		name.WriteByte('.')
	}

	if s.check(ast.IDENTIFIER) && s.peek().Lexeme == "as" {
		s.advance()

		alias, err := s.consume(ast.IDENTIFIER, "Expect name after 'as'.")
		if err != nil {
			return importDecl{}, err
		}

		as = alias
	}

	if _, err := s.consume(ast.SEMICOLON, "Expect ';' after import."); err != nil {
		return importDecl{}, err
	}

	return importDecl{tok: tok, name: name.String(), dots: dots, as: as}, nil
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

func braceDepths(tokens []ast.Token) []int {
	depth := make([]int, len(tokens))
	d := 0

	for i, t := range tokens {
		depth[i] = d

		switch {
		case t.TokenType == ast.LEFT_BRACE:
			d++
		case t.TokenType == ast.RIGHT_BRACE && d > 0:
			d--
		}
	}

	return depth
}

// synchronize discards tokens until it reaches a likely start of the next
// statement of the block being parsed, or its closing brace, skipping
// whatever braces the bad statement opens.
func (s *Parser) synchronize() {
	s.advance()

	for !s.isAtEnd() {
		if s.previous().TokenType == ast.SEMICOLON && s.depth[s.current-1] == s.blockDepth {
			return
		}

		// A statement on a new line after a block, as print(x) after an if.
		if prev := s.previous(); prev.TokenType == ast.RIGHT_BRACE && s.depth[s.current] == s.blockDepth && s.peek().Line > prev.Line {
			return
		}

		if s.depth[s.current] == s.blockDepth {
			switch t := s.peek().TokenType; t {
			case ast.FUN, ast.LET, ast.STRUCT, ast.IMPL, ast.IMPORT, ast.EXPORT, ast.MATCH:
				return
			case ast.RIGHT_BRACE:
				if s.blockDepth > 0 {
					return
				}
			default:
				if stmtKeyword(t) {
					return
				}
			}
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
