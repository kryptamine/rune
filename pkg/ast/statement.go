package ast

type StmtVisitor interface {
	VisitPrintStmt(printStmt *PrintStmt) error
	VisitExprStmt(exprStmt *ExprStmt) error
	VisitVarStmt(VarStmt *VarStmt) error
	VisitBlockStmt(blockStmt *BlockStmt) error
	VisitIfStmt(ifStmt *IfStmt) error
	VisitWhileStmt(whileStmt *WhileStmt) error
	VisitFunctionStmt(functionStmt *FunctionStmt) error
	VisitReturnStmt(returnStmt *ReturnStmt) error
}

type VarStmt struct {
	Initializer Expr
	Name        Token
}

func NewVarStmt(initializer Expr, name Token) Stmt {
	return &VarStmt{Initializer: initializer, Name: name}
}

type ReturnStmt struct {
	Value   Expr
	Keyword Token
}

func NewReturnStmt(value Expr, keyword Token) Stmt {
	return &ReturnStmt{Value: value, Keyword: keyword}
}

type PrintStmt struct {
	Expr Expr
}

func NewPrintStmt(expr Expr) Stmt {
	return &PrintStmt{Expr: expr}
}

type ExprStmt struct {
	Expr Expr
}

func NewExprStmt(expr Expr) Stmt {
	return &ExprStmt{Expr: expr}
}

type BlockStmt struct {
	Stmts []Stmt
}

func NewBlockStmt(stmts []Stmt) Stmt {
	return &BlockStmt{Stmts: stmts}
}

type IfStmt struct {
	Condition Expr
	Then      Stmt
	El        Stmt
}

func NewIfStmt(condition Expr, then Stmt, el Stmt) Stmt {
	return &IfStmt{Condition: condition, Then: then, El: el}
}

type WhileStmt struct {
	Condition Expr
	Body      Stmt
}

func NewWhileStmt(condition Expr, body Stmt) Stmt {
	return &WhileStmt{Condition: condition, Body: body}
}

type FunctionStmt struct {
	Name       Token
	Parameters []Token
	Body       []Stmt
}

func NewFunctionStmt(name Token, parameters []Token, body []Stmt) Stmt {
	return &FunctionStmt{Name: name, Parameters: parameters, Body: body}
}

type Stmt interface {
	Accept(v StmtVisitor) error
}

func (n *FunctionStmt) Accept(v StmtVisitor) error {
	return v.VisitFunctionStmt(n)
}

func (n *PrintStmt) Accept(v StmtVisitor) error {
	return v.VisitPrintStmt(n)
}

func (n *ExprStmt) Accept(v StmtVisitor) error {
	return v.VisitExprStmt(n)
}

func (n *VarStmt) Accept(v StmtVisitor) error {
	return v.VisitVarStmt(n)
}

func (n *BlockStmt) Accept(v StmtVisitor) error {
	return v.VisitBlockStmt(n)
}

func (n *IfStmt) Accept(v StmtVisitor) error {
	return v.VisitIfStmt(n)
}

func (n *WhileStmt) Accept(v StmtVisitor) error {
	return v.VisitWhileStmt(n)
}

func (n *ReturnStmt) Accept(v StmtVisitor) error {
	return v.VisitReturnStmt(n)
}
