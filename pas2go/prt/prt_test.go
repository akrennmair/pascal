package pat_test

import (
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"path/filepath"
	"testing"

	"github.com/akrennmair/pascal/parser"
	"github.com/stretchr/testify/require"
)

func init() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
}

func TestPascalRejectionTest(t *testing.T) {
	files, err := filepath.Glob("testdata/*.pas")
	require.NoError(t, err)

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			source, err := ioutil.ReadFile(file)
			require.NoError(t, err, "reading %s failed", file)

			_, err = parser.Parse(file, string(source))
			require.Error(t, err, "parser didn't return error for %s", file)
		})
	}
}
