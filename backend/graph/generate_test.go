package graph_test

import (
	"os"
	"testing"
)

func TestCodegenFilesExist(t *testing.T) {
	files := []string{
		"generated.go",
		"model/models_gen.go",
	}

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			info, err := os.Stat(f)
			if err != nil {
				t.Fatalf("file %s does not exist: %v", f, err)
			}
			if info.Size() == 0 {
				t.Fatalf("file %s is empty", f)
			}
		})
	}
}
