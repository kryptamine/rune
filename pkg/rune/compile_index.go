package rune

import (
	"rune/pkg/ast"
	"rune/pkg/value"
)

func compileArray(n *ast.ArrayExpr) evalFn {
	items := make([]evalFn, len(n.Items))
	for i, item := range n.Items {
		items[i] = compileExpr(item)
	}

	return func(p *Interpreter) (value.Value, error) {
		vals := make([]value.Value, len(items))

		for i, item := range items {
			v, err := item(p)
			if err != nil {
				return value.NilValue, err
			}

			vals[i] = v
		}

		return value.NewArray(vals), nil
	}
}

func compileObject(n *ast.ObjectExpr) evalFn {
	keys := make([]string, len(n.Pairs))
	vals := make([]evalFn, len(n.Pairs))

	for i, pair := range n.Pairs {
		keys[i] = pair.Key
		vals[i] = compileExpr(pair.Value)
	}

	return func(p *Interpreter) (value.Value, error) {
		obj := make(map[string]value.Value, len(keys))

		for i, val := range vals {
			v, err := val(p)
			if err != nil {
				return value.NilValue, err
			}

			obj[keys[i]] = v
		}

		return value.NewObject(obj), nil
	}
}

// compileProperty compiles obj.name, whose key is known up front.
func compileProperty(n *ast.IndexExpr) evalFn {
	target := compileExpr(n.Array)
	token, key := &n.Token, n.Token.Lexeme

	return func(p *Interpreter) (value.Value, error) {
		t, err := target(p)
		if err != nil {
			return value.NilValue, err
		}

		if t.Kind() != value.Object {
			return value.NilValue, propertyError("read", token, t)
		}

		if v, ok := t.Obj()[key]; ok {
			return v, nil
		}

		return value.NilValue, undefinedProperty(token, key)
	}
}

func compileIndex(n *ast.IndexExpr) evalFn {
	if n.Property {
		return compileProperty(n)
	}

	target := compileExpr(n.Array)
	index := compileExpr(n.Index)
	token := &n.Token

	return func(p *Interpreter) (value.Value, error) {
		t, err := target(p)
		if err != nil {
			return value.NilValue, err
		}

		i, err := index(p)
		if err != nil {
			return value.NilValue, err
		}

		if t.Kind() == value.Array && i.IsNumber() {
			arr := t.Arr()
			if idx := int(i.Num()); float64(idx) == i.Num() && idx >= 0 && idx < len(arr) {
				return arr[idx], nil
			}
		}

		return getIndex(token, t, i)
	}
}

// evalPlace evaluates the target and index of t[i] or t.name for a write.
func evalPlace(p *Interpreter, target, index evalFn, token *ast.Token, property bool) (value.Value, value.Value, error) {
	t, err := target(p)
	if err != nil {
		return value.NilValue, value.NilValue, err
	}

	if property && t.Kind() != value.Object {
		return value.NilValue, value.NilValue, propertyError("set", token, t)
	}

	i, err := index(p)

	return t, i, err
}

func compileSetIndex(n *ast.SetIndexExpr) evalFn {
	target := compileExpr(n.Array)
	index := compileExpr(n.Index)
	val := compileExpr(n.Value)
	token, op, property := &n.Token, n.Op, n.Property

	return func(p *Interpreter) (value.Value, error) {
		t, i, err := evalPlace(p, target, index, token, property)
		if err != nil {
			return value.NilValue, err
		}

		// For a compound assignment, read the old value before evaluating
		// the right-hand side, as "a[i] = a[i] op v" would.
		var old value.Value

		if op != nil {
			if old, err = getIndex(token, t, i); err != nil {
				return value.NilValue, err
			}
		}

		v, err := val(p)
		if err != nil {
			return value.NilValue, err
		}

		if op != nil {
			if v, err = binaryOp(op, old, v); err != nil {
				return value.NilValue, err
			}
		}

		if err := setIndex(token, t, i, v); err != nil {
			return value.NilValue, err
		}

		return v, nil
	}
}
