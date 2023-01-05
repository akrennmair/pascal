package pas2go

import (
	"bytes"
	"fmt"

	"github.com/akrennmair/pascal/parser"
)

func Transpile(ast *parser.AST) (string, error) {
	var buf bytes.Buffer

	if err := transpilerTemplate.ExecuteTemplate(&buf, "main", ast); err != nil {
		return "", fmt.Errorf("failed to generated Go source code: %w", err)
	}

	return buf.String(), nil
}
