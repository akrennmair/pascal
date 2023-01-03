package parser

func findBuiltinProcedure(name string) *Routine {
	for _, proc := range builtinProcedures {
		if proc.Name == name {
			return proc
		}
	}

	return nil
}

var builtinProcedures = []*Routine{
	{Name: "writeln", varargs: true},
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
