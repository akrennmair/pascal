package parser

import "fmt"

func FindBuiltinProcedure(name string) *Routine {
	return findBuiltinRoutine(builtinProcedures, name)
}

func FindBuiltinFunction(name string) *Routine {
	return findBuiltinRoutine(builtinFunctions, name)
}

func findBuiltinRoutine(list []*Routine, name string) *Routine {
	for _, proc := range list {
		if proc.Name == name {
			return proc
		}
	}

	return nil
}

var builtinProcedures = []*Routine{
	{
		Name: "writeln",
		validator: func(exprs []Expression) (DataType, error) {
			// TODO: implement
			return nil, nil
		},
	},
	{
		Name: "write",
		validator: func(exprs []Expression) (DataType, error) {
			// TODO: implement
			return nil, nil
		},
	},
	{
		Name:      "readln",
		validator: validateReadParameters,
	},
	{
		Name:      "read",
		validator: validateReadParameters,
	},
	{
		Name: "new",
		validator: func(exprs []Expression) (DataType, error) {
			if len(exprs) != 1 {
				return nil, fmt.Errorf("new requires exactly 1 argument of a pointer type, got %d arguments instead", len(exprs))
			}

			if _, ok := exprs[0].Type().(*PointerType); !ok {
				return nil, fmt.Errorf("new requires exactly 1 argument of a pointer type, got %s instead", exprs[0].Type().Type())
			}

			return nil, nil
		},
	},
	{
		Name: "dispose",
		validator: func(exprs []Expression) (DataType, error) {
			if len(exprs) != 1 {
				return nil, fmt.Errorf("dispose requires exactly 1 argument of a pointer type, got %d arguments instead", len(exprs))
			}

			if _, ok := exprs[0].Type().(*PointerType); !ok {
				return nil, fmt.Errorf("dispose requires exactly 1 argument of a pointer type, got %s instead", exprs[0].Type().Type())
			}

			return nil, nil
		},
	},
}

var builtinFunctions = []*Routine{
	{
		Name: "abs",
		validator: func(exprs []Expression) (DataType, error) {
			if len(exprs) != 1 {
				return nil, fmt.Errorf("abs requires exactly 1 argument of type integer or real, got %d arguments instead", len(exprs))
			}

			switch exprs[0].Type().(type) {
			case *IntegerType:
				return &IntegerType{}, nil
			case *RealType:
				return &RealType{}, nil
			}

			return nil, fmt.Errorf("abs requires exactly 1 argument of type integer or real, got %s instead", exprs[0].Type().Type())
		},
	},
	{
		Name: "arctan",
		FormalParameters: []*FormalParameter{
			{
				Name: "r",
				Type: &RealType{},
			},
		},
		ReturnType: &RealType{},
	},
	{
		Name: "cos",
		FormalParameters: []*FormalParameter{
			{
				Name: "r",
				Type: &RealType{},
			},
		},
		ReturnType: &RealType{},
	},
	{
		Name: "exp",
		FormalParameters: []*FormalParameter{
			{
				Name: "r",
				Type: &RealType{},
			},
		},
		ReturnType: &RealType{},
	},
	{
		Name: "frac",
		FormalParameters: []*FormalParameter{
			{
				Name: "r",
				Type: &RealType{},
			},
		},
		ReturnType: &RealType{},
	},
	{
		Name: "int",
		FormalParameters: []*FormalParameter{
			{
				Name: "r",
				Type: &RealType{},
			},
		},
		ReturnType: &RealType{},
	},
	{
		Name: "ln",
		FormalParameters: []*FormalParameter{
			{
				Name: "r",
				Type: &RealType{},
			},
		},
		ReturnType: &RealType{},
	},
	{
		Name:       "pi",
		ReturnType: &RealType{},
	},
	{
		Name: "sin",
		FormalParameters: []*FormalParameter{
			{
				Name: "r",
				Type: &RealType{},
			},
		},
		ReturnType: &RealType{},
	},
	{
		Name: "sqr",
		FormalParameters: []*FormalParameter{
			{
				Name: "r",
				Type: &RealType{},
			},
		},
		ReturnType: &RealType{},
	},
	{
		Name: "sqrt",
		FormalParameters: []*FormalParameter{
			{
				Name: "r",
				Type: &RealType{},
			},
		},
		ReturnType: &RealType{},
	},
	{
		Name: "trunc",
		FormalParameters: []*FormalParameter{
			{
				Name: "r",
				Type: &RealType{},
			},
		},
		ReturnType: &IntegerType{},
	},
	{
		Name: "round",
		FormalParameters: []*FormalParameter{
			{
				Name: "r",
				Type: &RealType{},
			},
		},
		ReturnType: &IntegerType{},
	},
	{
		Name: "chr",
		FormalParameters: []*FormalParameter{
			{
				Name: "i",
				Type: &IntegerType{},
			},
		},
		ReturnType: &CharType{},
	},
	{
		Name: "odd",
		FormalParameters: []*FormalParameter{
			{
				Name: "i",
				Type: &IntegerType{},
			},
		},
		ReturnType: &BooleanType{},
	},
}

func getBuiltinType(identifier string) DataType {
	switch identifier {
	case "boolean":
		return &BooleanType{}
	case "integer":
		return &IntegerType{}
	case "real":
		return &RealType{}
	case "char":
		return &CharType{}
	case "string":
		return &StringType{}
	}
	return nil
}

func getBuiltinEnumValues(identifier string) (idx int, typ DataType) {
	switch identifier {
	case "false":
		return 0, &BooleanType{}
	case "true":
		return 1, &BooleanType{}
	default:
		return 0, nil
	}
}

func validateReadParameters(exprs []Expression) (DataType, error) {
	for idx, e := range exprs {
		if !e.IsVariableExpr() {
			return nil, fmt.Errorf("expression %d is not a variable expression", idx)
		}
	}
	return nil, nil
}
