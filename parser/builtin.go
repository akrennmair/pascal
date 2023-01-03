package parser

func findBuiltinProcedure(name string) *routine {
	for _, proc := range builtinProcedures {
		if proc.Name == name {
			return proc
		}
	}

	return nil
}

var builtinProcedures = []*routine{
	{Name: "writeln", varargs: true},
}

func getBuiltinType(identifier string) dataType {
	switch identifier {
	case "boolean":
		return &booleanType{}
	case "integer":
		return &integerType{}
	case "real":
		return &realType{}
	case "char":
		return &charType{}
	case "string":
		return &arrayType{
			indexTypes: []dataType{
				&subrangeType{
					lowerBound: 1,
					upperBound: 255,
					typ:        &integerType{},
				},
			},
			elementType: &charType{},
			packed:      true,
		}
	}
	return nil
}
