package pat_test

import (
	"io/ioutil"
	"testing"

	"github.com/akrennmair/pascal/parser"
	"github.com/akrennmair/pascal/pas2go"
	"github.com/stretchr/testify/require"
)

func TestPascalAcceptanceTest(t *testing.T) {
	patSourceFile := "iso7185pat.pas"
	outputFile := "iso7185pat.out"

	source, err := ioutil.ReadFile(patSourceFile)
	require.NoError(t, err, "reading %s failed", patSourceFile)

	ast, err := parser.Parse(patSourceFile, string(source))
	require.NoError(t, err, "parsing %s failed", patSourceFile)

	goCode, err := pas2go.Transpile(ast)
	require.NoError(t, err, "transpiling %s failed", patSourceFile)

	ioutil.WriteFile(outputFile, []byte(goCode), 0644)

	t.Logf("Transpiler output is now available at %s", outputFile)
}
