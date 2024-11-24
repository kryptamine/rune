package main

type StmtVisitor interface {
	visitPrintStmt(printStmt *PrintStmt) error
	visitExprStmt(exprStmt *ExprStmt) error
	visitVarStmt(VarStmt *VarStmt) error
	visitBlockStmt(blockStmt *BlockStmt) error
}

type VarStmt struct {
	initializer Expr
	name        Token
}

type PrintStmt struct {
	expr Expr
}

type ExprStmt struct {
	expr Expr
}

type BlockStmt struct {
	stmts []Stmt
}

type Stmt interface {
	accept(v StmtVisitor) error
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

func (n *BlockStmt) accept(v StmtVisitor) error {
	return v.visitBlockStmt(n)
}
