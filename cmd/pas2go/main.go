package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/template"

	"github.com/akrennmair/pascal/parser"
)

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stdout, "usage: %s file.pas", os.Args[0])
		os.Exit(1)
	}

	sourceFile := flag.Arg(0)

	source, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		log.Fatalf("Reading file %s failed: %v", sourceFile, err)
	}

	ast, err := parser.Parse(sourceFile, string(source))
	if err != nil {
		log.Fatalf("Parsing %s failed: %v", sourceFile, err)
	}

	funcs := template.FuncMap{
		"toGoType":        toGoType,
		"sortTypeDefs":    sortTypeDefs,
		"constantLiteral": constantLiteral,
		"formalParams":    formalParams,
		"actualParams":    actualParams,
		"toExpr":          toExpr,
	}

	tmpl, err := template.New("").Funcs(funcs).Parse(sourceTemplate)
	if err != nil {
		log.Fatalf("Parsing template failed: %v", err)
	}

	var buf bytes.Buffer

	if err := tmpl.ExecuteTemplate(&buf, "main", ast); err != nil {
		log.Fatalf("Failed to generate source code: %v", err)
	}

	fmt.Print(buf.String())
}
