package rune

import (
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

// shapeStructs shapes the enums, or the structs and the cases of the
// enums, whose fields may name constants.
func (p *Resolver) shapeStructs(stmts []ast.Stmt, enums bool) {
	for _, stmt := range stmts {
		s, ok := stmt.(*ast.StructStmt)
		switch {
		case !ok || !s.Enum && enums:
			continue
		case s.Enum && !enums:
			p.defineCases(s)
			continue
		case s.Enum:
			s.Shape = p.shapeEnum(s)
		default:
			s.Shape = value.NewShape(s.Name.Lexeme, p.fields(s.Fields))
		}

		p.structs[p.mod.topLevel[s.Name.Lexeme]] = s.Shape
	}
}

// VisitStructStmt checks stmt is at the top level, whose structs
// resolveTopLevel has given their shapes.
func (p *Resolver) VisitStructStmt(stmt *ast.StructStmt) error {
	if stmt.Shape == nil {
		p.error(stmt.Name, helpers.If(stmt.Enum, "Enums", "Structs")+" must be declared at the top level.")
		return nil
	}

	stmt.Local = p.declare(ast.Binding{Name: stmt.Name})
	p.define(stmt.Name)

	return nil
}

// fields declares the fields of a struct or a case.
func (p *Resolver) fields(decls []ast.StructField) []value.Field {
	fields := make([]value.Field, len(decls))

	for i, f := range decls {
		name := p.intern(f.Name.Lexeme)

		var def value.Value
		if f.Default != nil {
			p.resolveExpr(f.Default)

			var ok bool
			if def, ok = p.constantOf(f.Default); !ok {
				p.error(f.DefaultAt, "Field default must be a constant: a literal, a const, an enum variant, or an array, object, map() or set() of them.")
			}
		}
		fields[i] = value.Field{Name: name, Default: def, Required: f.Default == nil}
	}

	return fields
}

func (p *Resolver) constantOf(e ast.Expr) (value.Value, bool) {
	if v, ok := literalOf(e); ok {
		return v, true
	}

	switch n := unwrap(e).(type) {
	case *ast.ArrayExpr:
		items := make([]value.Value, len(n.Items))
		for i, item := range n.Items {
			v, ok := p.constantOf(item)
			if !ok {
				return value.NilValue, false
			}
			items[i] = v
		}
		return value.NewArray(items), true
	case *ast.ObjectExpr:
		fields := value.NewObj(len(n.Pairs))
		for _, pair := range n.Pairs {
			v, ok := p.constantOf(pair.Value)
			if !ok {
				return value.NilValue, false
			}
			fields.Set(pair.Key, v)
		}
		return value.NewObject(fields), true
	case *ast.IndexExpr:
		return p.variantOf(n)
	case *ast.CallExpr:
		callee, ok := n.Callee.(*ast.VarExpr)
		if !ok || len(n.Args) > 0 || callee.Module != nil || !p.isBuiltin(callee.Name.Lexeme) {
			return value.NilValue, false
		}
		switch callee.Name.Lexeme {
		case "map":
			return value.NewMap(value.NewHashMap(0)), true
		case "set":
			return value.NewSet(value.NewHashMap(0)), true
		}
	}

	return value.NilValue, false
}

// intern returns the one string of a field or method name.
func (p *Resolver) intern(name string) string {
	if s, ok := p.fieldNames[name]; ok {
		return s
	}

	p.fieldNames[name] = name

	return name
}

func (p *Resolver) VisitStructExpr(expr *ast.StructExpr) (value.Value, error) {
	for _, pair := range expr.Pairs {
		p.resolveExpr(pair.Value)
	}

	p.resolveExpr(expr.Type)

	shape := p.structType(expr.Type)
	if shape == nil {
		return value.NilValue, nil
	}

	if shape.Enum {
		p.error(expr.Type.Name, notStruct(shape))
		return value.NilValue, nil
	}

	expr.Shape = shape
	given := make([]bool, len(shape.Fields))
	spread := false

	for i, pair := range expr.Pairs {
		if ast.IsSpread(pair.Value) {
			spread = true
			continue
		}

		slot := shape.Index(pair.Key.Lexeme)
		if slot < 0 {
			p.error(pair.Key, errors.NoField(shape.Name, pair.Key.Lexeme))
			continue
		}

		expr.Pairs[i].Slot = slot
		given[slot] = true
	}

	for i, f := range shape.Fields {
		if _, optional := shape.Default(i); !given[i] && !optional && !spread {
			p.error(expr.Type.Name, errors.MissingField(shape.Name, f))
		}
	}

	return value.NilValue, nil
}
