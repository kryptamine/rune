package ast

import (
	"fmt"
	"rune/pkg/value"
)

type ExprVisitor interface {
	VisitBinaryExpr(binaryExpr *BinaryExpr) (value.Value, error)
	VisitLiteralExpr(literalExpr *LiteralExpr) (value.Value, error)
	VisitGroupingExpr(literalExpr *GroupingExpr) (value.Value, error)
	VisitUnaryExpr(UnaryExpr *UnaryExpr) (value.Value, error)
	VisitVarExpr(varExpr *VarExpr) (value.Value, error)
	VisitAssignExpr(assignExpr *AssignExpr) (value.Value, error)
	VisitLogicalExpr(logicalExpr *LogicalExpr) (value.Value, error)
	VisitCallExpr(callExpr *CallExpr) (value.Value, error)
	VisitArrayExpr(arrayExpr *ArrayExpr) (value.Value, error)
	VisitIndexExpr(indexExpr *IndexExpr) (value.Value, error)
	VisitSetIndexExpr(setIndexExpr *SetIndexExpr) (value.Value, error)
	VisitObjectExpr(objectExpr *ObjectExpr) (value.Value, error)
	VisitFunctionExpr(functionExpr *FunctionExpr) (value.Value, error)
	VisitUpdateExpr(updateExpr *UpdateExpr) (value.Value, error)
	VisitRangeExpr(rangeExpr *RangeExpr) (value.Value, error)
	VisitStructExpr(structExpr *StructExpr) (value.Value, error)
	VisitIsExpr(isExpr *IsExpr) (value.Value, error)
	VisitMatchExpr(matchExpr *MatchExpr) (value.Value, error)
	VisitInterpolationExpr(interpolationExpr *InterpolationExpr) (value.Value, error)
	VisitOptionalExpr(optionalExpr *OptionalExpr) (value.Value, error)
	VisitSpreadExpr(spreadExpr *SpreadExpr) (value.Value, error)
}

type Expr interface {
	Accept(v ExprVisitor) (value.Value, error)
}

type BinaryExpr struct {
	Left     Expr
	Right    Expr
	Operator Token
}

func NewBinaryExpr(left Expr, right Expr, operator Token) Expr {
	return &BinaryExpr{Left: left, Right: right, Operator: operator}
}

func (n *BinaryExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitBinaryExpr(n)
}

func (n *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %v %v)", n.Operator.Lexeme, n.Left, n.Right)
}

// Local is a variable's declaration, shared by every reference the
// resolver binds to it, built-ins included. Other names have none.
type Local struct {
	// Captured is set when a nested function refers to the variable.
	Captured bool
	// TopLevel is set for a declaration outside any block or function.
	TopLevel bool
}

type AssignExpr struct {
	Name  Token
	Value Expr
	Local *Local
}

func NewAssignExpr(name Token, value Expr) Expr {
	return &AssignExpr{Name: name, Value: value}
}

type UnaryExpr struct {
	Right    Expr
	Operator Token
}

func NewUnaryExpr(right Expr, operator Token) Expr {
	return &UnaryExpr{Right: right, Operator: operator}
}

type LiteralExpr struct {
	TokenType TokenType
	Value     value.Value
}

// NewLiteralExpr accepts a plain Go value (nil, bool, float64 or string).
func NewLiteralExpr(tokenType TokenType, v any) Expr {
	return &LiteralExpr{TokenType: tokenType, Value: value.FromGo(v)}
}

// VarExpr is a name; Module is set when it is qualified, as in m.Point.
type VarExpr struct {
	Module *Token
	Name   Token
	Local  *Local
}

func NewVarExpr(name Token) Expr {
	return &VarExpr{Name: name}
}

type LogicalExpr struct {
	Left  Expr
	Right Expr
	Op    Token
}

func NewLogicalExpr(left Expr, right Expr, op Token) Expr {
	return &LogicalExpr{Left: left, Right: right, Op: op}
}

type GroupingExpr struct {
	Expr Expr
}

func NewGroupingExpr(expr Expr) Expr {
	return &GroupingExpr{Expr: expr}
}

// CallExpr is callee(args); Optional is set for callee?.(args). For a
// method call, Writable is set when the receiver can take the new self a
// mutating method leaves.
type CallExpr struct {
	Token    Token
	Callee   Expr
	Args     []Expr
	Optional bool
	Writable bool
}

type ArrayExpr struct {
	TokenType TokenType
	Items     []Expr
}

func NewArrayExpr(tokenType TokenType, items []Expr) Expr {
	return &ArrayExpr{TokenType: tokenType, Items: items}
}

// IndexExpr is arr[i], or obj.name when Property is set.
type IndexExpr struct {
	Token    Token
	Array    Expr
	Index    Expr
	Property bool
	// Optional is set for arr?.[i] and obj?.name.
	Optional bool
	// Member is the built-in a namespace member names, set by the resolver.
	Member *value.Value
	// Export is the variable a module member names, set by the resolver.
	Export *VarExpr
}

func NewPropertyExpr(object Expr, name Token, optional bool) Expr {
	return &IndexExpr{Array: object, Index: NewLiteralExpr(STRING, name.Lexeme), Token: name, Property: true, Optional: optional}
}

// SetIndexExpr assigns to an array element or object property. For a
// compound assignment (arr[i] += v) Op is the binary operator to apply to
// the old value; otherwise it is nil. Array and Index are evaluated once.
type SetIndexExpr struct {
	Token    Token
	Array    Expr
	Index    Expr
	Value    Expr
	Op       *Token
	Property bool
}

func NewSetIndexExpr(target *IndexExpr, value Expr, op *Token) Expr {
	return &SetIndexExpr{Array: target.Array, Index: target.Index, Value: value, Token: target.Token, Op: op, Property: target.Property}
}

// UpdateExpr is ++x, x++, --x or x--. Target is a *VarExpr or *IndexExpr.
type UpdateExpr struct {
	Target   Expr
	Operator Token
	Prefix   bool
}

func NewUpdateExpr(target Expr, operator Token, prefix bool) Expr {
	return &UpdateExpr{Target: target, Operator: operator, Prefix: prefix}
}

func (n *UpdateExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitUpdateExpr(n)
}

func (n *UpdateExpr) String() string {
	if n.Prefix {
		return fmt.Sprintf("(%s %v)", n.Operator.Lexeme, n.Target)
	}

	return fmt.Sprintf("(%v %s)", n.Target, n.Operator.Lexeme)
}

// FunctionExpr is an anonymous function: fun (a, b) { ... }. Its Decl has
// an empty name and is never declared in any scope.
type FunctionExpr struct {
	Decl *FunctionStmt
}

func NewFunctionExpr(decl *FunctionStmt) Expr {
	return &FunctionExpr{Decl: decl}
}

func (n *FunctionExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitFunctionExpr(n)
}

// StructPair is a "field: value" entry of a struct literal; the resolver
// sets Slot, the field's index in the declaration.
type StructPair struct {
	Key   Token
	Value Expr
	Slot  int
}

// StructExpr is a struct literal, Point { x: 1 } or m.Point { x: 1 };
// the resolver sets Shape.
type StructExpr struct {
	Type  *VarExpr
	Pairs []StructPair
	Shape *value.Shape
}

func (n *StructExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitStructExpr(n)
}

// IsExpr is value is Type.
type IsExpr struct {
	Value Expr
	Type  Expr
	Token Token
}

func NewIsExpr(v, typ Expr, token Token) Expr {
	return &IsExpr{Value: v, Type: typ, Token: token}
}

func (n *IsExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitIsExpr(n)
}

// InterpolationExpr is a string with "{expr}" parts: string literals and
// the expressions between them, in order.
type InterpolationExpr struct {
	Parts []Expr
}

func (n *InterpolationExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitInterpolationExpr(n)
}

// OptionalExpr is a chain with a ?. in it: when the value before one is
// nil, the whole chain is nil.
type OptionalExpr struct {
	Expr Expr
}

func (n *OptionalExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitOptionalExpr(n)
}

// SpreadExpr is ...expr, an item of an array or object literal or an argument.
type SpreadExpr struct {
	Token Token
	Expr  Expr
}

func (n *SpreadExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitSpreadExpr(n)
}

// IsSpread reports whether e is a SpreadExpr.
func IsSpread(e Expr) bool {
	_, ok := e.(*SpreadExpr)
	return ok
}

// ObjectPair is a single "key: value" entry of an object literal, or a
// spread, whose Value is a *SpreadExpr.
type ObjectPair struct {
	Key   string
	Value Expr
}

type ObjectExpr struct {
	TokenType TokenType
	// Pairs keep source order so values are evaluated left to right.
	Pairs []ObjectPair
}

func NewObjectExpr(tokenType TokenType, pairs []ObjectPair) Expr {
	return &ObjectExpr{TokenType: tokenType, Pairs: pairs}
}

func (n *ObjectExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitObjectExpr(n)
}

func (n *ArrayExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitArrayExpr(n)
}

func (n *IndexExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitIndexExpr(n)
}

func (n *LiteralExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitLiteralExpr(n)
}

func (n *LiteralExpr) String() string {
	if n.Value.IsNil() {
		return "nil"
	}

	if n.Value.IsNumber() {
		l := n.Value.Num()
		// Check if the float is an integer value
		if l == float64(int64(l)) {
			// Print with one decimal place (e.g., 10.0 instead of 10)
			return fmt.Sprintf("%.1f", l)
		} else {
			return fmt.Sprintf("%g", l)
		}
	}

	switch n.Value.Kind() {
	case value.String:
		return n.Value.Str()
	case value.Bool:
		return fmt.Sprint(n.Value.Bool())
	}

	return fmt.Sprintf("%v", n.Value.Fn())
}

func (n *VarExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitVarExpr(n)
}

func (n *GroupingExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitGroupingExpr(n)
}

func (n *GroupingExpr) String() string {
	return fmt.Sprintf("(group %v)", n.Expr)
}

func (n *UnaryExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitUnaryExpr(n)
}

func (n *UnaryExpr) String() string {
	return fmt.Sprintf("(%s %v)", n.Operator.Lexeme, n.Right)
}

func (n *AssignExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitAssignExpr(n)
}

func (n *AssignExpr) String() string {
	return fmt.Sprintf("(assign %v %v)", n.Name, n.Value)
}

func (n *LogicalExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitLogicalExpr(n)
}

func (n *CallExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitCallExpr(n)
}

func (n *SetIndexExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitSetIndexExpr(n)
}

// RangeExpr is start..end or start..=end, with an optional step (nil means 1).
type RangeExpr struct {
	Start     Expr
	End       Expr
	Step      Expr
	Operator  Token
	Inclusive bool
}

func NewRangeExpr(start Expr, end Expr, step Expr, operator Token) Expr {
	return &RangeExpr{Start: start, End: end, Step: step, Operator: operator, Inclusive: operator.TokenType == DOT_DOT_EQUAL}
}

func (n *RangeExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitRangeExpr(n)
}

func (n *RangeExpr) String() string {
	if n.Step != nil {
		return fmt.Sprintf("(%s %v %v step %v)", n.Operator.Lexeme, n.Start, n.End, n.Step)
	}

	return fmt.Sprintf("(%s %v %v)", n.Operator.Lexeme, n.Start, n.End)
}
