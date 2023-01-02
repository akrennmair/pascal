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
	}

	b := &block{
		variables: []*variable{
			{
				Name: "a",
				Type: &integerType{},
			},
			{
				Name: "b",
				Type: &integerType{},
			},
			{
				Name: "c",
				Type: &integerType{},
			},
			{
				Name: "i",
				Type: &integerType{},
			},
			{
				Name: "j",
				Type: &integerType{},
			},
			{
				Name: "s",
				Type: &arrayType{
					indexTypes:  []dataType{&subrangeType{1, 10, &integerType{}}},
					elementType: &integerType{},
				},
			},
			{
				Name: "data",
				Type: &arrayType{
					indexTypes:  []dataType{&subrangeType{1, 10, &integerType{}}},
					elementType: &integerType{},
				},
			},
			{
				Name: "matrix",
				Type: &arrayType{
					indexTypes:  []dataType{&subrangeType{1, 3, &integerType{}}, &subrangeType{1, 3, &integerType{}}},
					elementType: &integerType{},
				},
			},
			{
				Name: "result",
				Type: &realType{},
			},
			{
				Name: "x",
				Type: &integerType{},
			},
			{
				Name: "ptr",
				Type: &pointerType{typ: &integerType{}},
			},
			{
				Name: "d",
				Type: &booleanType{},
			},
			{
				Name: "e",
				Type: &booleanType{},
			},
			{
				Name: "foo",
				Type: &recordType{
					fields: []*recordField{
						{
							Identifiers: []string{"data"},
							Type:        &integerType{},
						},
					},
				},
			},
			{
				Name: "l",
				Type: &realType{},
			},
			{
				Name: "m",
				Type: &realType{},
			},
			{
				Name: "o",
				Type: &realType{},
			},
		},
		functions: []*routine{
			{
				Name: "length",
				FormalParameters: []*formalParameter{
					{
						Name: "arr",
						Type: &arrayType{
							indexTypes:  []dataType{&subrangeType{lowerBound: 1, upperBound: 10, typ: &integerType{}}},
							elementType: &integerType{},
						},
					},
				},
				ReturnType: &integerType{},
			},
		},
	}

	for _, tt := range testData {
		t.Run(tt.Name, func(t *testing.T) {
			p := NewParser(tt.Name, tt.Expr)

			var (
				err  error
				expr expression
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
