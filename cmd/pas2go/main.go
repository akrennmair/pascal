package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/akrennmair/pascal/parser"
)

func main() {
	var outputFile string

	flag.StringVar(&outputFile, "o", "", "if non-empty, where the output will be written to")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stdout, "usage: %s [-o output.go] file.pas", os.Args[0])
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

	goSource, err := transpile(ast)
	if err != nil {
		log.Fatalf("Transpiling %s failed: %v", sourceFile, err)
	}

	if outputFile == "" {
		fmt.Print(goSource)
	} else {
		if err := ioutil.WriteFile(outputFile, []byte(goSource), 0644); err != nil {
			log.Fatalf("Couldn't write to output file %s: %v", outputFile, err)
		}
	}
}

func transpile(ast *parser.AST) (string, error) {
	var buf bytes.Buffer

	if err := transpilerTemplate.ExecuteTemplate(&buf, "main", ast); err != nil {
		return "", fmt.Errorf("failed to generated Go source code: %w", err)
	}

	return buf.String(), nil
}
