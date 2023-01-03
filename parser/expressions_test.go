package parser

import (
	"testing"
)

func TestParseExpressions(t *testing.T) {
	testData := []struct {
		Name      string
		Expr      string
		ExpectErr bool
	}{
		{Name: "simple variable", Expr: "a", ExpectErr: false},
		{Name: "comparison of two variables", Expr: "a = b", ExpectErr: false},
		{Name: "comparison of variable with integer literal", Expr: "a = 1", ExpectErr: false},
		{Name: "comparison of variable with negative integer literal", Expr: "a = -23", ExpectErr: false},
		{Name: "comparison negative integer literal with variable", Expr: "-42 <> a", ExpectErr: false},
		{Name: "more complex expression", Expr: "not ((a >= 0) and (a <= 9))", ExpectErr: false},
		{Name: "two ANDed comparisons", Expr: "(a = 1) and (b = 2)", ExpectErr: false},
		{Name: "two ORed comparisons", Expr: "(a = 1) or (b = 2)", ExpectErr: false},
		{Name: "addition expression", Expr: "a + b - c", ExpectErr: false},
		{Name: "integer multiplication expression", Expr: "a * b div c", ExpectErr: false},
		{Name: "integer multiplication expression with float divide", Expr: "a * b / c", ExpectErr: true},
		{Name: "real multiplication expression", Expr: "l * m / o", ExpectErr: false},
		{Name: "real multiplication expression with integer divide", Expr: "l * m div o", ExpectErr: true},
		{Name: "subexpressions", Expr: "(a = 2) or (b <> 3)", ExpectErr: false},
		{Name: "more complex logical expression", Expr: "d and e or (d and not e)", ExpectErr: false},
		{Name: "more complex logical expression with comparisons", Expr: "(a <= 3) and (b > -23) or ((a <= -3) and not (b >= 400))", ExpectErr: false},
		//{Name: "condition from check whether string is a number", Expr: "not ((s[i] >= '0') and (s[i] <= '9'))", ExpectErr: false}, // add back in whenever we support char
		{Name: "condition that is a function call", Expr: "length(data)", ExpectErr: false},
		{Name: "condition with record variable", Expr: "foo.data <> 23", ExpectErr: false},
		{Name: "condition with floating point literal", Expr: "result <= 3.1415538", ExpectErr: false},
		{Name: "condition with floating point literal with scale factor", Expr: "result >= 2e-9", ExpectErr: false},
		{Name: "condition with floating point literal with scale factor and dot", Expr: "result >= 1.234e 5", ExpectErr: false},
		{Name: "set literal", Expr: "x in [ 5, 10, 23 ]", ExpectErr: false},
		{Name: "indexed variable with multiple dimensions", Expr: "matrix[i, j] = 23", ExpectErr: false},
		{Name: "pointer comparison with nil", Expr: "ptr <> nil", ExpectErr: false},
		{Name: "addition of two literals with one negative number", Expr: "5 + -3", ExpectErr: false},
		{Name: "less than comparison of string variable with string literal", Expr: "str < 'abc'", ExpectErr: false},
		{Name: "less than comparison of string literal with string variable", Expr: "'abc' < str", ExpectErr: false},
	}

	b := &Block{
		Variables: []*Variable{
			{
				Name: "a",
				Type: &IntegerType{},
			},
			{
				Name: "b",
				Type: &IntegerType{},
			},
			{
				Name: "c",
				Type: &IntegerType{},
			},
			{
				Name: "i",
				Type: &IntegerType{},
			},
			{
				Name: "j",
				Type: &IntegerType{},
			},
			{
				Name: "s",
				Type: &ArrayType{
					IndexTypes:  []DataType{&SubrangeType{1, 10, &IntegerType{}}},
					ElementType: &IntegerType{},
				},
			},
			{
				Name: "data",
				Type: &ArrayType{
					IndexTypes:  []DataType{&SubrangeType{1, 10, &IntegerType{}}},
					ElementType: &IntegerType{},
				},
			},
			{
				Name: "matrix",
				Type: &ArrayType{
					IndexTypes:  []DataType{&SubrangeType{1, 3, &IntegerType{}}, &SubrangeType{1, 3, &IntegerType{}}},
					ElementType: &IntegerType{},
				},
			},
			{
				Name: "result",
				Type: &RealType{},
			},
			{
				Name: "x",
				Type: &IntegerType{},
			},
			{
				Name: "ptr",
				Type: &PointerType{Type_: &IntegerType{}},
			},
			{
				Name: "d",
				Type: &BooleanType{},
			},
			{
				Name: "e",
				Type: &BooleanType{},
			},
			{
				Name: "foo",
				Type: &RecordType{
					Fields: []*RecordField{
						{
							Identifier: "data",
							Type:       &IntegerType{},
						},
					},
				},
			},
			{
				Name: "l",
				Type: &RealType{},
			},
			{
				Name: "m",
				Type: &RealType{},
			},
			{
				Name: "o",
				Type: &RealType{},
			},
			{
				Name: "str",
				Type: &StringType{},
			},
		},
		Functions: []*Routine{
			{
				Name: "length",
				FormalParameters: []*FormalParameter{
					{
						Name: "arr",
						Type: &ArrayType{
							IndexTypes:  []DataType{&SubrangeType{LowerBound: 1, UpperBound: 10, Type_: &IntegerType{}}},
							ElementType: &IntegerType{},
						},
					},
				},
				ReturnType: &IntegerType{},
			},
		},
	}

	for _, tt := range testData {
		t.Run(tt.Name, func(t *testing.T) {
			p := newParser(tt.Name, tt.Expr)

			var (
				err  error
				expr Expression
			)

			func() {
				defer p.recover(&err)
				expr = p.parseExpression(b)
			}()

			if tt.ExpectErr {
				if err == nil {
					t.Errorf("Expected error, but got no error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
				if p.peek().typ != itemEOF {
					t.Errorf("Parser has not consumed all tokens, stopped at %s", p.peek())
				}
			}
			t.Logf("expr %s = %s", tt.Expr, expr)
		})
	}
}
