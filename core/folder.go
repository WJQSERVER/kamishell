package core

import "math"

// Fold performs constant folding on the parsed AST.
// It evaluates pure literal sub-expressions at parse time and replaces
// them with their computed results. This reduces runtime overhead for
// expressions like `3 + 4 * 2` (→ 11) or `"a" + "b"` (→ "ab").
func Fold(program *Program) {
	for i, stmt := range program.Statements {
		program.Statements[i] = foldStatement(stmt)
	}
}

func foldStatement(stmt Statement) Statement {
	switch s := stmt.(type) {
	case *ExpressionStatement:
		s.Expression = foldExpression(s.Expression)
		return s
	case *PrintStatement:
		s.Expression = foldExpression(s.Expression)
		return s
	case *AssignStatement:
		s.Value = foldExpression(s.Value)
		return s
	case *VarStatement:
		s.Value = foldExpression(s.Value)
		return s
	case *IfStatement:
		s.Condition = foldExpression(s.Condition)
		if s.Consequence != nil {
			foldBlock(s.Consequence)
		}
		if s.Alternative != nil {
			foldBlock(s.Alternative)
		}
		return s
	case *ForStatement:
		if s.Init != nil {
			s.Init = foldStatement(s.Init)
		}
		s.Condition = foldExpression(s.Condition)
		if s.Post != nil {
			s.Post = foldStatement(s.Post)
		}
		if s.Consequence != nil {
			foldBlock(s.Consequence)
		}
		return s
	case *ReturnStatement:
		for i, rv := range s.ReturnValues {
			s.ReturnValues[i] = foldExpression(rv)
		}
		return s
	case *SwitchStatement:
		s.Tag = foldExpression(s.Tag)
		for i := range s.Cases {
			for j, v := range s.Cases[i].Values {
				s.Cases[i].Values[j] = foldExpression(v)
			}
			if s.Cases[i].Body != nil {
				foldBlock(s.Cases[i].Body)
			}
		}
		return s
	case *PipeStatement:
		for i, cmd := range s.Commands {
			s.Commands[i] = foldStatement(cmd)
		}
		return s
	case *RedirectStatement:
		s.Source = foldStatement(s.Source)
		s.Target = foldExpression(s.Target)
		return s
	case *LogicalStatement:
		s.Left = foldStatement(s.Left)
		s.Right = foldStatement(s.Right)
		return s
	case *BackgroundStatement:
		s.Stmt = foldStatement(s.Stmt)
		return s
	case *GoStatement:
		s.Node = foldNode(s.Node)
		return s
	case *ExecStatement:
		if s.CommandStr != nil {
			s.CommandStr = foldExpression(s.CommandStr)
		}
		for i, arg := range s.Args {
			s.Args[i] = foldExpression(arg)
		}
		return s
	case *WaitStatement:
		s.Timeout = foldExpression(s.Timeout)
		return s
	case *MethodCallBlockStatement:
		s.Object = foldExpression(s.Object)
		if s.Body != nil {
			foldBlock(s.Body)
		}
		return s
	case *FunctionStatement:
		if s.Body != nil {
			foldBlock(s.Body)
		}
		return s
	case *PointerAssignStatement:
		s.Value = foldExpression(s.Value)
		return s
	default:
		return stmt
	}
}

func foldNode(node Node) Node {
	switch n := node.(type) {
	case *BlockStatement:
		foldBlock(n)
		return n
	case *ExpressionStatement:
		n.Expression = foldExpression(n.Expression)
		return n
	case *CommandStatement:
		for i, arg := range n.Arguments {
			n.Arguments[i] = foldExpression(arg)
		}
		return n
	default:
		return node
	}
}

func foldBlock(block *BlockStatement) {
	if block == nil {
		return
	}
	for i, stmt := range block.Statements {
		block.Statements[i] = foldStatement(stmt)
	}
}

func foldExpression(expr Expression) Expression {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *InfixExpression:
		return foldInfix(e)
	case *PrefixExpression:
		return foldPrefix(e)
	case *CallExpression:
		e.Function = foldExpression(e.Function)
		for i, arg := range e.Arguments {
			e.Arguments[i] = foldExpression(arg)
		}
		return e
	case *IndexExpression:
		e.Left = foldExpression(e.Left)
		e.Index = foldExpression(e.Index)
		return e
	case *ArrayLiteral:
		for i, el := range e.Elements {
			e.Elements[i] = foldExpression(el)
		}
		return e
	case *FunctionLiteral:
		if e.Body != nil {
			foldBlock(e.Body)
		}
		return e
	case *GoExpression:
		e.Node = foldNode(e.Node)
		return e
	default:
		// Literals, Identifiers, etc. — no folding needed
		return expr
	}
}

func foldInfix(e *InfixExpression) Expression {
	e.Left = foldExpression(e.Left)
	e.Right = foldExpression(e.Right)

	left := e.Left
	right := e.Right
	op := e.Operator

	// String + String → String (concatenation)
	if op == "+" {
		if ls, ok := left.(*StringLiteral); ok {
			if rs, ok := right.(*StringLiteral); ok {
				if ls.Obj != nil && rs.Obj != nil {
					return &StringLiteral{
						Token: ls.Token,
						Value: ls.Value + rs.Value,
						Obj:   &String{Value: ls.Value + rs.Value},
					}
				}
			}
		}
	}

	// Integer op Integer → Integer
	if li, ok := left.(*IntegerLiteral); ok && li.Err == "" {
		if ri, ok := right.(*IntegerLiteral); ok && ri.Err == "" {
			return foldIntInt(li, ri, op)
		}
	}

	// Float op Float → Float
	if lf, ok := left.(*FloatLiteral); ok && lf.Err == "" {
		if rf, ok := right.(*FloatLiteral); ok && rf.Err == "" {
			return foldFloatFloat(lf, rf, op)
		}
	}

	// Integer op Float → Float (type promotion)
	if li, ok := left.(*IntegerLiteral); ok && li.Err == "" {
		if rf, ok := right.(*FloatLiteral); ok && rf.Err == "" {
			return foldFloatFloat(&FloatLiteral{Value: float64(li.Value)}, rf, op)
		}
	}

	// Float op Integer → Float (type promotion)
	if lf, ok := left.(*FloatLiteral); ok && lf.Err == "" {
		if ri, ok := right.(*IntegerLiteral); ok && ri.Err == "" {
			return foldFloatFloat(lf, &FloatLiteral{Value: float64(ri.Value)}, op)
		}
	}

	// Boolean == Boolean, Boolean != Boolean
	if li, ok := left.(*BooleanLiteral); ok {
		if ri, ok := right.(*BooleanLiteral); ok {
			switch op {
			case "==":
				return &BooleanLiteral{Value: li.Value == ri.Value}
			case "!=":
				return &BooleanLiteral{Value: li.Value != ri.Value}
			}
		}
	}

	// String == String, String != String, etc.
	if ls, ok := left.(*StringLiteral); ok && ls.Obj != nil {
		if rs, ok := right.(*StringLiteral); ok && rs.Obj != nil {
			lv, rv := ls.Value, rs.Value
			switch op {
			case "==":
				return &BooleanLiteral{Value: lv == rv}
			case "!=":
				return &BooleanLiteral{Value: lv != rv}
			case ">":
				return &BooleanLiteral{Value: lv > rv}
			case "<":
				return &BooleanLiteral{Value: lv < rv}
			case ">=":
				return &BooleanLiteral{Value: lv >= rv}
			case "<=":
				return &BooleanLiteral{Value: lv <= rv}
			}
		}
	}

	// nil == nil, nil != nil
	if _, ok := left.(*NilLiteral); ok {
		if _, ok := right.(*NilLiteral); ok {
			switch op {
			case "==":
				return &BooleanLiteral{Value: true}
			case "!=":
				return &BooleanLiteral{Value: false}
			}
		}
	}

	return e
}

func foldIntInt(li, ri *IntegerLiteral, op string) Expression {
	lv, rv := li.Value, ri.Value
	switch op {
	case "+":
		return &IntegerLiteral{Value: lv + rv, Obj: getIntegerObject(lv + rv)}
	case "-":
		return &IntegerLiteral{Value: lv - rv, Obj: getIntegerObject(lv - rv)}
	case "*":
		return &IntegerLiteral{Value: lv * rv, Obj: getIntegerObject(lv * rv)}
	case "/":
		if rv == 0 {
			return &IntegerLiteral{Value: 0, Err: "division by zero"}
		}
		return &IntegerLiteral{Value: lv / rv, Obj: getIntegerObject(lv / rv)}
	case "%":
		if rv == 0 {
			return &IntegerLiteral{Value: 0, Err: "division by zero"}
		}
		return &IntegerLiteral{Value: lv % rv, Obj: getIntegerObject(lv % rv)}
	case "==":
		return &BooleanLiteral{Value: lv == rv}
	case "!=":
		return &BooleanLiteral{Value: lv != rv}
	case ">":
		return &BooleanLiteral{Value: lv > rv}
	case "<":
		return &BooleanLiteral{Value: lv < rv}
	case ">=":
		return &BooleanLiteral{Value: lv >= rv}
	case "<=":
		return &BooleanLiteral{Value: lv <= rv}
	default:
		return &IntegerLiteral{Value: lv, Obj: li.Obj}
	}
}

func foldFloatFloat(lf, rf *FloatLiteral, op string) Expression {
	lv, rv := lf.Value, rf.Value
	switch op {
	case "+":
		return &FloatLiteral{Value: lv + rv, Obj: &Float{Value: lv + rv}}
	case "-":
		return &FloatLiteral{Value: lv - rv, Obj: &Float{Value: lv - rv}}
	case "*":
		return &FloatLiteral{Value: lv * rv, Obj: &Float{Value: lv * rv}}
	case "/":
		if rv == 0 {
			return &FloatLiteral{Value: 0, Err: "division by zero"}
		}
		return &FloatLiteral{Value: lv / rv, Obj: &Float{Value: lv / rv}}
	case "%":
		if rv == 0 {
			return &FloatLiteral{Value: 0, Err: "division by zero"}
		}
		return &FloatLiteral{Value: math.Mod(lv, rv), Obj: &Float{Value: math.Mod(lv, rv)}}
	case "==":
		return &BooleanLiteral{Value: lv == rv}
	case "!=":
		return &BooleanLiteral{Value: lv != rv}
	case ">":
		return &BooleanLiteral{Value: lv > rv}
	case "<":
		return &BooleanLiteral{Value: lv < rv}
	case ">=":
		return &BooleanLiteral{Value: lv >= rv}
	case "<=":
		return &BooleanLiteral{Value: lv <= rv}
	default:
		return &FloatLiteral{Value: lv, Obj: lf.Obj}
	}
}

func foldPrefix(e *PrefixExpression) Expression {
	e.Right = foldExpression(e.Right)

	switch e.Operator {
	case "!":
		if bl, ok := e.Right.(*BooleanLiteral); ok {
			return &BooleanLiteral{Value: !bl.Value}
		}
	case "-":
		if il, ok := e.Right.(*IntegerLiteral); ok && il.Err == "" {
			return &IntegerLiteral{Value: -il.Value, Obj: getIntegerObject(-il.Value)}
		}
		if fl, ok := e.Right.(*FloatLiteral); ok && fl.Err == "" {
			return &FloatLiteral{Value: -fl.Value, Obj: &Float{Value: -fl.Value}}
		}
	}

	return e
}
