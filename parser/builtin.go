package parser

import (
	"fmt"
	"math"
)

// FindBuiltinProcedure returns the _builtin_ procedure with the
// provided name. If no such procedure exists, it returns nil.
func FindBuiltinProcedure(name string) *Routine {
	return findBuiltinRoutine(builtinProcedures, name)
}

// FindBuiltinFunction returns the _builtin_ function with the
// provided name. If no such function exists, it returns nil.
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
				return nil, fmt.Errorf("new requires exactly 1 argument of a pointer type, got %s instead", exprs[0].Type().TypeString())
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
				return nil, fmt.Errorf("dispose requires exactly 1 argument of a pointer type, got %s instead", exprs[0].Type().TypeString())
			}

			return nil, nil
		},
	},
	{
		Name: "inc",
		validator: func(exprs []Expression) (DataType, error) {
			switch len(exprs) {
			case 2:
				if _, ok := exprs[1].Type().(*IntegerType); !ok {
					return nil, fmt.Errorf("inc: second argument has to be an integer expression")
				}
				fallthrough
			case 1:
				if _, isIntegerType := exprs[0].Type().(*IntegerType); !isIntegerType || !exprs[0].IsVariableExpr() {
					return nil, fmt.Errorf("inc: first argument has to be an integer variable")
				}
			default:
				return nil, fmt.Errorf("inc: wrong amount of arguments")
			}
			return nil, nil
		},
	},
	{
		Name: "dec",
		validator: func(exprs []Expression) (DataType, error) {
			switch len(exprs) {
			case 2:
				if _, ok := exprs[1].Type().(*IntegerType); !ok {
					return nil, fmt.Errorf("dec: second argument has to be an integer expression")
				}
				fallthrough
			case 1:
				if _, isIntegerType := exprs[0].Type().(*IntegerType); !isIntegerType || !exprs[0].IsVariableExpr() {
					return nil, fmt.Errorf("dec: first argument has to be an integer variable")
				}
			default:
				return nil, fmt.Errorf("dec: wrong amount of arguments")
			}
			return nil, nil
		},
	},
	{
		Name: "rewrite",
		validator: func(exprs []Expression) (DataType, error) {
			if len(exprs) != 1 {
				return nil, fmt.Errorf("rewrite: need exactly 1 argument of file type")
			}

			if _, ok := exprs[0].Type().(*FileType); ok {
				return nil, nil
			}

			return nil, fmt.Errorf("rewrite: need exactly 1 argument of file type, got %s instead", exprs[0].Type().TypeString())
		},
	},
	{
		Name: "reset",
		validator: func(exprs []Expression) (DataType, error) {
			if len(exprs) != 1 {
				return nil, fmt.Errorf("reset: need exactly 1 argument of file type")
			}

			if _, ok := exprs[0].Type().(*FileType); ok {
				return nil, nil
			}

			return nil, fmt.Errorf("reset: need exactly 1 argument of file type, got %s instead", exprs[0].Type().TypeString())
		},
	},
	{
		Name: "unpack",
		validator: func(exprs []Expression) (DataType, error) {
			// TODO: implement
			return nil, nil
		},
	},
	{
		Name: "pack",
		validator: func(exprs []Expression) (DataType, error) {
			// TODO: implement
			return nil, nil
		},
	},
	{
		Name: "get",
		validator: func(exprs []Expression) (DataType, error) {
			// TODO: implement
			return nil, nil
		},
	},
	{
		Name: "put",
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
			case *SubrangeType:
				return exprs[0].Type(), nil
			}

			return nil, fmt.Errorf("abs requires exactly 1 argument of type integer or real, got %s instead", exprs[0].Type().TypeString())
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
		ReturnType: charTypeDef.Type,
	},
	{
		Name: "odd",
		FormalParameters: []*FormalParameter{
			{
				Name: "i",
				Type: &IntegerType{},
			},
		},
		ReturnType: booleanTypeDef.Type,
	},
	{
		Name: "ord",
		validator: func(exprs []Expression) (DataType, error) {
			if len(exprs) != 1 {
				return nil, fmt.Errorf("ord requires exactly 1 argument of type enum or char, got %d arguments instead", len(exprs))
			}

			if IsCharType(exprs[0].Type()) {
				return &IntegerType{}, nil
			}

			switch exprs[0].Type().(type) {
			case *EnumType:
				return &IntegerType{}, nil
			case *SubrangeType:
				return &IntegerType{}, nil
			}

			if _, isStringLiteral := exprs[0].(*StringExpr); isStringLiteral {
				return &IntegerType{}, nil
			}

			return nil, fmt.Errorf("ord requires exactly 1 argument of type enum or char, got %s instead", exprs[0].Type().TypeString())
		},
	},
	{
		Name: "succ",
		validator: func(exprs []Expression) (DataType, error) {
			if len(exprs) != 1 {
				return nil, fmt.Errorf("succ requires exactly 1 argument of type enum or integer, got %d arguments instead", len(exprs))
			}

			switch exprs[0].Type().(type) {
			case *IntegerType:
				return exprs[0].Type(), nil
			case *EnumType:
				return exprs[0].Type(), nil
			case *SubrangeType:
				return exprs[0].Type(), nil
			}

			return nil, fmt.Errorf("succ requires exactly 1 argument of type enum or integer, got %s instead", exprs[0].Type().TypeString())
		},
	},
	{
		Name: "pred",
		validator: func(exprs []Expression) (DataType, error) {
			if len(exprs) != 1 {
				return nil, fmt.Errorf("pred requires exactly 1 argument of type enum or integer, got %d arguments instead", len(exprs))
			}

			switch exprs[0].Type().(type) {
			case *IntegerType:
				return exprs[0].Type(), nil
			case *EnumType:
				return exprs[0].Type(), nil
			case *SubrangeType:
				return exprs[0].Type(), nil
			}

			return nil, fmt.Errorf("pred requires exactly 1 argument of type enum or integer, got %s instead", exprs[0].Type().TypeString())
		},
	},
	{
		Name: "eof",
		validator: func(exprs []Expression) (DataType, error) {
			if len(exprs) != 1 {
				return nil, fmt.Errorf("eof requires exactly 1 argument of file type, got %d arguments instead", len(exprs))
			}

			switch exprs[0].Type().(type) {
			case *FileType:
				return booleanTypeDef.Type, nil
			}

			return nil, fmt.Errorf("eof requires exactly 1 argument of file type, got %s instead", exprs[0].Type().TypeString())
		},
	},
	{
		Name: "eoln",
		validator: func(exprs []Expression) (DataType, error) {
			if len(exprs) != 1 {
				return nil, fmt.Errorf("eoln requires exactly 1 argument of file type, got %d arguments instead", len(exprs))
			}

			switch exprs[0].Type().(type) {
			case *FileType:
				return booleanTypeDef.Type, nil
			}

			return nil, fmt.Errorf("eoln requires exactly 1 argument of file type, got %s instead", exprs[0].Type().TypeString())
		},
	},
}

func getBuiltinType(identifier string) DataType {
	switch identifier {
	case "integer":
		return &IntegerType{}
	case "real":
		return &RealType{}
	case "string":
		return &StringType{}
	case "char":
		return charTypeDef.Type
	case "boolean":
		return booleanTypeDef.Type
	case "text":
		return textTypeDef.Type
	}
	return nil
}

func validateReadParameters(exprs []Expression) (DataType, error) {
	for idx, e := range exprs {
		if !e.IsVariableExpr() {
			return nil, fmt.Errorf("expression %d is not a variable expression", idx)
		}
	}
	return nil, nil
}

var booleanTypeDef = &TypeDefinition{
	Name: "boolean",
	Type: &EnumType{
		Identifiers: []string{"false", "true"},
		name:        "boolean",
	},
}

var charTypeDef = &TypeDefinition{
	Name: "char",
	Type: &SubrangeType{
		LowerBound: 0,
		UpperBound: 255,
		name:       "",
		Type_:      &IntegerType{},
	},
}

var textTypeDef = &TypeDefinition{
	Name: "text",
	Type: &FileType{
		ElementType: charTypeDef.Type,
		name:        "text",
	},
}

// IsBooleanType returns true if the provided type is the boolean type, false otherwise.
func IsBooleanType(dt DataType) bool {
	return booleanTypeDef.Type.Equals(dt)
}

// IsCharType returns true if the provided type is the char type, false otherwise.
func IsCharType(dt DataType) bool {
	return charTypeDef.Type.Equals(dt)
}

var builtinBlock = &Block{
	Constants: []*ConstantDefinition{
		{
			Name:  "maxint",
			Value: &IntegerLiteral{Value: math.MaxInt},
		},
	},
	Types: []*TypeDefinition{
		booleanTypeDef,
		charTypeDef,
		textTypeDef,
	},
}
