package parser

import (
	"fmt"
	"strings"
)

type Expression interface {
	String() string
	Type() DataType
	IsVariableExpr() bool
	Reduce() Expression // reduce nested expressions to innermost single expression
}

func isRelationalOperator(typ itemType) bool {
	return typ == itemEqual ||
		typ == itemNotEqual ||
		typ == itemLess ||
		typ == itemLessEqual ||
		typ == itemGreater ||
		typ == itemGreaterEqual ||
		typ == itemIn
}

type RelationalOperator string

const (
	opEqual        RelationalOperator = "="
	opNotEqual     RelationalOperator = "<>"
	opLess         RelationalOperator = "<"
	opLessEqual    RelationalOperator = "<="
	opGreater      RelationalOperator = ">"
	opGreaterEqual RelationalOperator = ">="
	opIn           RelationalOperator = "in"
)

var itemTypeRelOps = map[itemType]RelationalOperator{
	itemEqual:        opEqual,
	itemNotEqual:     opNotEqual,
	itemLess:         opLess,
	itemLessEqual:    opLessEqual,
	itemGreater:      opGreater,
	itemGreaterEqual: opGreaterEqual,
	itemIn:           opIn,
}

func itemTypeToRelationalOperator(typ itemType) RelationalOperator {
	op, ok := itemTypeRelOps[typ]
	if !ok {
		return RelationalOperator(fmt.Sprintf("INVALID(%d)", typ))
	}
	return op
}

type RelationalExpr struct {
	Left     Expression
	Operator RelationalOperator
	Right    Expression
}

func (e *RelationalExpr) String() string {
	return fmt.Sprintf("relation<%s %s %s>", e.Left, e.Operator, e.Right)
}

func (e *RelationalExpr) Type() DataType {
	return &BooleanType{}
}

func (e *RelationalExpr) IsVariableExpr() bool {
	return false
}

func (e *RelationalExpr) Reduce() Expression {
	return &RelationalExpr{
		Left:     e.Left.Reduce(),
		Operator: e.Operator,
		Right:    e.Right.Reduce(),
	}
}

// TODO: has this ever been used? Add test for it.
type minusExpr struct {
	expr *TermExpr
}

func (e *minusExpr) String() string {
	return fmt.Sprintf("minus<%s>", e.expr)
}

func (e *minusExpr) Type() DataType {
	return e.expr.Type()
}

func (e *minusExpr) IsVariableExpr() bool {
	return false
}

func (e *minusExpr) Reduce() Expression {
	return e
}

func isAdditionOperator(typ itemType) bool {
	return typ == itemSign || typ == itemOr
}

type SimpleExpr struct {
	Sign  string
	First Expression
	Next  []*Addition
}

func (e *SimpleExpr) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "simple-expr:<%s %s", e.Sign, e.First)
	for _, add := range e.Next {
		fmt.Fprintf(&buf, " %s %s", add.Operator, add.Term)
	}
	fmt.Fprint(&buf, ">")
	return buf.String()
}

func (e *SimpleExpr) Type() DataType {
	return e.First.Type()
}

func (e *SimpleExpr) IsVariableExpr() bool {
	if e.Sign != "" || len(e.Next) > 0 {
		return false
	}

	return e.First.IsVariableExpr()
}

func (e *SimpleExpr) Reduce() Expression {
	if len(e.Next) == 0 && (e.Sign == "" || e.Sign == "+") {
		return e.First.Reduce()
	}
	ne := &SimpleExpr{
		Sign:  e.Sign,
		First: e.First.Reduce(),
	}

	for _, add := range e.Next {
		ne.Next = append(ne.Next, &Addition{
			Operator: add.Operator,
			Term:     add.Term.Reduce(),
		})
	}

	return ne
}

type AdditionOperator string

const (
	OperatorAdd      AdditionOperator = "+"
	OperatorSubtract AdditionOperator = "-"
	OperatorOr       AdditionOperator = "or"
)

func tokenToAdditionOperator(t item) AdditionOperator {
	if t.typ == itemSign && (t.val == "+" || t.val == "-") {
		return AdditionOperator(t.val)
	}
	if t.typ == itemOr {
		return OperatorOr
	}
	return AdditionOperator(fmt.Sprintf("INVALID(%+v)", t))
}

type Addition struct {
	Operator AdditionOperator
	Term     Expression
}

type TermExpr struct {
	First Expression
	Next  []*Multiplication
}

func (e *TermExpr) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "term-expr:<%s", e.First)
	for _, mul := range e.Next {
		fmt.Fprintf(&buf, " %s %s", mul.Operator, mul.Factor)
	}
	fmt.Fprint(&buf, ">")
	return buf.String()
}

func (e *TermExpr) Type() DataType {
	return e.First.Type()
}

func (e *TermExpr) IsVariableExpr() bool {
	if len(e.Next) > 0 {
		return false
	}

	return e.First.IsVariableExpr()
}

func (e *TermExpr) Reduce() Expression {
	if len(e.Next) == 0 {
		return e.First.Reduce()
	}

	ne := &TermExpr{
		First: e.First.Reduce(),
	}

	for _, mul := range ne.Next {
		ne.Next = append(ne.Next, &Multiplication{
			Operator: mul.Operator,
			Factor:   mul.Factor.Reduce(),
		})
	}
	return e
}

type Multiplication struct {
	Operator MultiplicationOperator
	Factor   Expression
}

type MultiplicationOperator string

const (
	OperatorMultiply    MultiplicationOperator = "*"
	OperatorFloatDivide MultiplicationOperator = "/"
	OperatorDivide      MultiplicationOperator = "div"
	OperatorModulo      MultiplicationOperator = "mod"
	OperatorAnd         MultiplicationOperator = "and"
)

var itemTypMulOp = map[itemType]MultiplicationOperator{
	itemMultiply:    OperatorMultiply,
	itemFloatDivide: OperatorFloatDivide,
	itemDiv:         OperatorDivide,
	itemMod:         OperatorModulo,
	itemAnd:         OperatorAnd,
}

func itemTypeToMultiplicationOperator(typ itemType) MultiplicationOperator {
	op, ok := itemTypMulOp[typ]
	if !ok {
		return MultiplicationOperator(fmt.Sprintf("INVALID(%+v)", typ))
	}
	return op
}

func isMultiplicationOperator(typ itemType) bool {
	return typ == itemMultiply ||
		typ == itemFloatDivide ||
		typ == itemDiv ||
		typ == itemMod ||
		typ == itemAnd
}

type ConstantExpr struct {
	Name  string
	Type_ DataType
}

func (e *ConstantExpr) String() string {
	return fmt.Sprintf("constant:<%s : %s>", e.Name, e.Type_.Type())
}

func (e *ConstantExpr) Type() DataType {
	return e.Type_
}

func (e *ConstantExpr) IsVariableExpr() bool {
	return false
}

func (e *ConstantExpr) Reduce() Expression {
	return e
}

type VariableExpr struct {
	Name  string
	Type_ DataType
}

func (e *VariableExpr) String() string {
	return fmt.Sprintf("variable:<%s>", e.Name)
}

func (e *VariableExpr) Type() DataType {
	return e.Type_
}

func (e *VariableExpr) IsVariableExpr() bool {
	return true
}

func (e *VariableExpr) Reduce() Expression {
	return e
}

type IntegerExpr struct {
	Value int64
}

func (e *IntegerExpr) String() string {
	return fmt.Sprintf("int:<%d>", e.Value)
}

func (e *IntegerExpr) Type() DataType {
	return &IntegerType{}
}

func (e *IntegerExpr) IsVariableExpr() bool {
	return false
}

func (e *IntegerExpr) Reduce() Expression {
	return e
}

type RealExpr struct {
	Minus       bool
	BeforeComma string
	AfterComma  string
	ScaleFactor int
}

func (e *RealExpr) String() string {
	sign := ""
	if e.Minus {
		sign = "-"
	}
	return fmt.Sprintf("real:<%s%s.%se%d>", sign, e.BeforeComma, e.AfterComma, e.ScaleFactor)
}

func (e *RealExpr) Type() DataType {
	return &RealType{}
}

func (e *RealExpr) IsVariableExpr() bool {
	return false
}

func (e *RealExpr) Reduce() Expression {
	return e
}

type StringExpr struct {
	Value string
}

func (e *StringExpr) String() string {
	return fmt.Sprintf("str:<%q>", e.Value)
}

func (e *StringExpr) Type() DataType {
	return &StringType{}
}

func (e *StringExpr) IsVariableExpr() bool {
	return false
}

func (e *StringExpr) Reduce() Expression {
	return e
}

func (e *StringExpr) IsCharLiteral() bool {
	// TODO: solve this neater.
	return len(e.Value) == 1 ||
		(len(e.Value) == 3 && e.Value[0] == '\'' && e.Value[2] == '\'') ||
		e.Value == "''''"
}

type NilExpr struct{}

func (e *NilExpr) String() string {
	return "nil"
}

func (e *NilExpr) Type() DataType {
	return &PointerType{Type_: nil} // nil means it's compatible with any type
}

func (e *NilExpr) IsVariableExpr() bool {
	return false
}

func (e *NilExpr) Reduce() Expression {
	return e
}

type NotExpr struct {
	Expr Expression
}

func (e *NotExpr) String() string {
	return fmt.Sprintf("not:<%s>", e.Expr)
}

func (e *NotExpr) Type() DataType {
	return e.Expr.Type()
}

func (e *NotExpr) IsVariableExpr() bool {
	return false
}

func (e *NotExpr) Reduce() Expression {
	return e
}

type SetExpr struct {
	Elements []Expression
}

func (e *SetExpr) String() string {
	var buf strings.Builder
	fmt.Fprint(&buf, "set-expr:<")
	for idx, elem := range e.Elements {
		if idx > 0 {
			fmt.Fprint(&buf, " ")
		}
		fmt.Fprintf(&buf, "%s", elem)
	}
	fmt.Fprint(&buf, ">")
	return buf.String()
}

func (e *SetExpr) Type() DataType {
	return &SetType{
		ElementType: e.Elements[0].Type(),
	}
}

func (e *SetExpr) IsVariableExpr() bool {
	return false
}

func (e *SetExpr) Reduce() Expression {
	return e
}

type SubExpr struct {
	Expr Expression
}

func (e *SubExpr) String() string {
	return fmt.Sprintf("sub-expr:<%s>", e.Expr)
}

func (e *SubExpr) Type() DataType {
	return e.Expr.Type()
}

func (e *SubExpr) IsVariableExpr() bool {
	return e.Expr.IsVariableExpr() // TODO: check whether this is correct.
}

func (e *SubExpr) Reduce() Expression {
	return e.Expr.Reduce()
}

type IndexedVariableExpr struct {
	Expr       Expression // an expression of type *arrayType
	Type_      DataType
	IndexExprs []Expression
}

func (e *IndexedVariableExpr) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "indexed-variable-expr:<(%s)[", e.Expr.String())
	for idx, expr := range e.IndexExprs {
		if idx > 0 {
			fmt.Fprint(&buf, ", ")
		}
		fmt.Fprintf(&buf, "%s", expr)
	}
	fmt.Fprint(&buf, "]>")
	return buf.String()
}

func (e *IndexedVariableExpr) Type() DataType {
	return e.Type_
}

func (e *IndexedVariableExpr) IsVariableExpr() bool {
	return true
}

func (e *IndexedVariableExpr) Reduce() Expression {
	ne := &IndexedVariableExpr{
		Expr:  e.Expr.Reduce(),
		Type_: e.Type_,
	}
	for _, ie := range e.IndexExprs {
		ne.IndexExprs = append(ne.IndexExprs, ie.Reduce())
	}
	return ne
}

type FunctionCallExpr struct {
	Name         string
	Type_        DataType
	ActualParams []Expression
}

func (e *FunctionCallExpr) String() string {
	var buf strings.Builder

	fmt.Fprintf(&buf, "function-call-expr:<%s(", e.Name)
	for idx, param := range e.ActualParams {
		if idx > 0 {
			fmt.Fprintf(&buf, ", ")
		}
		fmt.Fprint(&buf, param)
	}
	fmt.Fprint(&buf, ")>")
	return buf.String()
}

func (e *FunctionCallExpr) Type() DataType {
	return e.Type_
}

func (e *FunctionCallExpr) IsVariableExpr() bool {
	return false
}

func (e *FunctionCallExpr) Reduce() Expression {
	ne := &FunctionCallExpr{
		Name:  e.Name,
		Type_: e.Type_,
	}

	for _, pe := range ne.ActualParams {
		ne.ActualParams = append(ne.ActualParams, pe.Reduce())
	}

	return ne
}

type FieldDesignatorExpr struct {
	Expr  Expression
	Field string
	Type_ DataType
}

func (e *FieldDesignatorExpr) String() string {
	return fmt.Sprintf("field-designator-expr:<%s.%s>", e.Expr, e.Field)
}

func (e *FieldDesignatorExpr) Type() DataType {
	return e.Type_
}

func (e *FieldDesignatorExpr) IsVariableExpr() bool {
	return true
}

func (e *FieldDesignatorExpr) Reduce() Expression {
	return &FieldDesignatorExpr{
		Expr:  e.Expr.Reduce(),
		Field: e.Field,
		Type_: e.Type_,
	}
}

type EnumValueExpr struct {
	Name  string
	Value int
	Type_ DataType
}

func (e *EnumValueExpr) String() string {
	return fmt.Sprintf("enum-value-expr:<%s %d of type %s>", e.Name, e.Value, e.Type_.Type())
}

func (e *EnumValueExpr) Type() DataType {
	return e.Type_
}

func (e *EnumValueExpr) IsVariableExpr() bool {
	return true
}

func (e *EnumValueExpr) Reduce() Expression {
	return e
}

type DerefExpr struct {
	Expr Expression
}

func (e *DerefExpr) String() string {
	return fmt.Sprintf("deref-expr:<%s>", e.Expr)
}

func (e *DerefExpr) Type() DataType {
	t, ok := e.Expr.Type().(*PointerType)
	if !ok {
		panic("derefExpr was created with expression not of pointer type")
	}
	return t.Type_
}

func (e *DerefExpr) IsVariableExpr() bool {
	return e.Expr.IsVariableExpr()
}

func (e *DerefExpr) Reduce() Expression {
	return &DerefExpr{
		Expr: e.Expr.Reduce(),
	}
}

type FormatExpr struct {
	Expr          Expression
	Width         Expression
	DecimalPlaces Expression
}

func (e *FormatExpr) String() string {
	var buf strings.Builder

	buf.WriteString("format-expr:<")
	buf.WriteString(e.Expr.String())
	if e.Width != nil {
		buf.WriteString(":")
		buf.WriteString(e.Width.String())
	}
	if e.DecimalPlaces != nil {
		buf.WriteString(":")
		buf.WriteString(e.DecimalPlaces.String())
	}
	buf.WriteString(">")
	return buf.String()
}

func (e *FormatExpr) Type() DataType {
	return e.Expr.Type()
}

func (e *FormatExpr) IsVariableExpr() bool {
	return e.Expr.IsVariableExpr()
}

func (e *FormatExpr) Reduce() Expression {
	return e
}
