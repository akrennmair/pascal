package pas2go

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/akrennmair/pascal/parser"
)

func Transpile(ast *parser.AST) (string, error) {
	var buf bytes.Buffer

	if err := transpilerTemplate.ExecuteTemplate(&buf, "main", ast); err != nil {
		return "", fmt.Errorf("failed to generated Go source code: %w", err)
	}

	//fmt.Printf("transpile: src = %s\n", buf.String())

	cmd := exec.Command("gofmt", "-s")
	cmd.Stdin = &buf
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("running gofmt failed: %w (%s)", err, string(output))
	}

	return string(output), nil
}
