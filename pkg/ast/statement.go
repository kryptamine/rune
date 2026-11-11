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
	VisitTryStmt(tryStmt *TryStmt) error
	VisitThrowStmt(throwStmt *ThrowStmt) error
	VisitBreakStmt(breakStmt *BreakStmt) error
	VisitContinueStmt(continueStmt *ContinueStmt) error
	VisitForInStmt(forInStmt *ForInStmt) error
}

// Binding is a declared name; Mutable is set for names declared with "mut".
type Binding struct {
	Name    Token
	Mutable bool
}

// VarStmt declares a variable. Local is set by the resolver for variables
// declared inside a scope, together with their Slot in it; for globals Slot
// is filled lazily by the interpreter.
type VarStmt struct {
	Binding
	Initializer Expr
	Local       bool
	Slot        int
}

func NewVarStmt(initializer Expr, binding Binding) Stmt {
	return &VarStmt{Initializer: initializer, Binding: binding, Slot: Unresolved}
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
// Pooled (here and on other scopes): no closure is created inside.
type BlockStmt struct {
	Stmts   []Stmt
	EnvSize int
	Pooled  bool
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

// WhileStmt is a while loop, or a desugared C-style for loop: Increment
// (nil for while) runs after every iteration, including one cut short by
// continue.
type WhileStmt struct {
	Condition Expr
	Body      Stmt
	Increment Expr
}

func NewWhileStmt(condition Expr, body Stmt, increment Expr) Stmt {
	return &WhileStmt{Condition: condition, Body: body, Increment: increment}
}

// ForInStmt iterates over an array (elements), a string (characters) or an
// object (keys, in sorted order). With two variables, Key gets the index
// (or the object key) and Value the element. Each iteration gets a fresh
// scope of EnvSize slots: Key (if any) first, then Value.
type ForInStmt struct {
	Keyword  Token
	Key      *Binding
	Value    Binding
	Iterable Expr
	Body     Stmt
	EnvSize  int
	Pooled   bool
}

func NewForInStmt(keyword Token, key *Binding, val Binding, iterable Expr, body Stmt) Stmt {
	return &ForInStmt{Keyword: keyword, Key: key, Value: val, Iterable: iterable, Body: body}
}

type BreakStmt struct {
	Keyword Token
}

func NewBreakStmt(keyword Token) Stmt {
	return &BreakStmt{Keyword: keyword}
}

type ContinueStmt struct {
	Keyword Token
}

func NewContinueStmt(keyword Token) Stmt {
	return &ContinueStmt{Keyword: keyword}
}

// FunctionStmt declares a function. Local and Slot describe the function's
// own name as in VarStmt; EnvSize is the number of locals in its body scope
// (parameters first, in order).
type FunctionStmt struct {
	Name       Token
	Parameters []Binding
	Body       []Stmt
	Local      bool
	Slot       int
	EnvSize    int
	Pooled     bool
}

func NewFunctionStmt(name Token, parameters []Binding, body []Stmt) Stmt {
	return &FunctionStmt{Name: name, Parameters: parameters, Body: body, Slot: Unresolved}
}

// TryStmt runs Body and, if it fails with a runtime error or a throw, runs
// Handler. Param is the optional catch variable (nil without one); it takes
// slot 0 of the handler's scope. HandlerEnvSize is the number of locals in
// that scope; zero means the handler runs in the enclosing environment.
type TryStmt struct {
	Body           Stmt
	Param          *Token
	Handler        []Stmt
	HandlerEnvSize int
	HandlerPooled  bool
}

func NewTryStmt(body Stmt, param *Token, handler []Stmt) Stmt {
	return &TryStmt{Body: body, Param: param, Handler: handler}
}

type ThrowStmt struct {
	Value   Expr
	Keyword Token
}

func NewThrowStmt(value Expr, keyword Token) Stmt {
	return &ThrowStmt{Value: value, Keyword: keyword}
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

func (n *TryStmt) Accept(v StmtVisitor) error {
	return v.VisitTryStmt(n)
}

func (n *ThrowStmt) Accept(v StmtVisitor) error {
	return v.VisitThrowStmt(n)
}

func (n *BreakStmt) Accept(v StmtVisitor) error {
	return v.VisitBreakStmt(n)
}

func (n *ContinueStmt) Accept(v StmtVisitor) error {
	return v.VisitContinueStmt(n)
}

func (n *ForInStmt) Accept(v StmtVisitor) error {
	return v.VisitForInStmt(n)
}
