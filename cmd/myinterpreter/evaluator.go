package main

type Interpreter struct{}

func (p *Interpreter) visitBinaryExpr(node *BinaryExpr) any {
	return nil
}

func (p *Interpreter) visitLiteralExpr(node *LiteralExpr) any {
	return node.value
}

func (p *Interpreter) visitGroupingExpr(node *GroupingExpr) any {
	return nil
}

func (p *Interpreter) visitUnaryExpr(node *UnaryExpr) any {
	return nil
}
