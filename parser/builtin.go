package parser

import "fmt"

func findBuiltinProcedure(name string) *Routine {
	return findBuiltinRoutine(builtinProcedures, name)
}

func findBuiltinFunction(name string) *Routine {
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
