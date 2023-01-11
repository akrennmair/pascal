package pas2go

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/akrennmair/pascal/parser"
	"github.com/stretchr/testify/require"
)

func TestTranspile(t *testing.T) {
	pascalFiles, err := filepath.Glob("testdata/*.pas")
	require.NoError(t, err)

	for _, pascalFile := range pascalFiles {
		t.Run(pascalFile, func(t *testing.T) {
			writeMode := false

			fileContent, err := ioutil.ReadFile(pascalFile)
			require.NoError(t, err)

			goldenFile := pascalFile + ".golden"

			goldenFileContent, err := ioutil.ReadFile(goldenFile)
			if err != nil {
				writeMode = true
			}

			ast, err := parser.Parse(pascalFile, string(fileContent))
			require.NoError(t, err, "parsing source file failed")

			//fmt.Printf("ast = %s\n", spew.Sdump(ast))

			goSource, err := Transpile(ast)
			require.NoError(t, err, "transpile failed")

			if writeMode {
				t.Logf("Writing transpiler output to missing golden file %s", goldenFile)
				require.NoError(t, ioutil.WriteFile(goldenFile, []byte(goSource), 0644))
			} else {
				require.Equal(t, string(goldenFileContent), goSource, "transpiler output doesn't match golden file")
			}
		})
	}
}
