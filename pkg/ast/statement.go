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

// VarStmt declares a variable. Local is set by the resolver for variables
// declared inside a scope, together with their Slot in it; for globals Slot
// is filled lazily by the interpreter.
type VarStmt struct {
	Initializer Expr
	Name        Token
	Local       bool
	Slot        int
}

func NewVarStmt(initializer Expr, name Token) Stmt {
	return &VarStmt{Initializer: initializer, Name: name, Slot: Unresolved}
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

// BlockStmt is a braced block. EnvSize is the number of locals it declares;
// a block with none runs in the enclosing environment.
type BlockStmt struct {
	Stmts   []Stmt
	EnvSize int
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

// FunctionStmt declares a function. Local and Slot describe the function's
// own name as in VarStmt; EnvSize is the number of locals in its body scope
// (parameters first, in order).
type FunctionStmt struct {
	Name       Token
	Parameters []Token
	Body       []Stmt
	Local      bool
	Slot       int
	EnvSize    int
}

func NewFunctionStmt(name Token, parameters []Token, body []Stmt) Stmt {
	return &FunctionStmt{Name: name, Parameters: parameters, Body: body, Slot: Unresolved}
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
