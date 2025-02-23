package solus

type StmtVisitor interface {
	visitPrintStmt(printStmt *PrintStmt) error
	visitExprStmt(exprStmt *ExprStmt) error
	visitVarStmt(VarStmt *VarStmt) error
	visitBlockStmt(blockStmt *BlockStmt) error
	visitIfStmt(ifStmt *IfStmt) error
	visitWhileStmt(whileStmt *WhileStmt) error
	visitFunctionStmt(functionStmt *FunctionStmt) error
	visitReturnStmt(returnStmt *ReturnStmt) error
}

type VarStmt struct {
	initializer Expr
	name        Token
}

func NewVarStmt(initializer Expr, name Token) Stmt {
	return &VarStmt{initializer: initializer, name: name}
}

type ReturnStmt struct {
	value   Expr
	keyword Token
}

func NewReturnStmt(value Expr, keyword Token) Stmt {
	return &ReturnStmt{value: value, keyword: keyword}
}

type PrintStmt struct {
	expr Expr
}

func NewPrintStmt(expr Expr) Stmt {
	return &PrintStmt{expr: expr}
}

type ExprStmt struct {
	expr Expr
}

func NewExprStmt(expr Expr) Stmt {
	return &ExprStmt{expr: expr}
}

type BlockStmt struct {
	stmts []Stmt
}

func NewBlockStmt(stmts []Stmt) Stmt {
	return &BlockStmt{stmts: stmts}
}

type IfStmt struct {
	condition Expr
	then      Stmt
	el        Stmt
}

func NewIfStmt(condition Expr, then Stmt, el Stmt) Stmt {
	return &IfStmt{condition: condition, then: then, el: el}
}

type WhileStmt struct {
	condition Expr
	body      Stmt
}

type FunctionStmt struct {
	name       Token
	parameters []Token
	body       []Stmt
}

func NewFunctionStmt(name Token, parameters []Token, body []Stmt) Stmt {
	return &FunctionStmt{name: name, parameters: parameters, body: body}
}

type Stmt interface {
	accept(v StmtVisitor) error
}

func (n *FunctionStmt) accept(v StmtVisitor) error {
	return v.visitFunctionStmt(n)
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

func (n *IfStmt) accept(v StmtVisitor) error {
	return v.visitIfStmt(n)
}

func (n *WhileStmt) accept(v StmtVisitor) error {
	return v.visitWhileStmt(n)
}

func (n *ReturnStmt) accept(v StmtVisitor) error {
	return v.visitReturnStmt(n)
}
