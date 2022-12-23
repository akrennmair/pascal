package parser

func findBuiltinProcedure(name string) *procedure {
	for _, proc := range builtinProcedures {
		if proc.Name == name {
			return proc
		}
	}

	return nil
}

var builtinProcedures = []*procedure{
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
	case "string":
		return &stringType{}
	}
	return nil
}
