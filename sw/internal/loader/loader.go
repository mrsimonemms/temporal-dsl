package loader

import (
	"os"
	"path/filepath"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/serverlessworkflow/sdk-go/v3/parser"
)

func FromFile(path string) (*model.Workflow, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	switch ext := filepath.Ext(path); ext {
	case ".yaml", ".yml":
		return parser.FromYAMLSource(data)
	case ".json":
		return parser.FromJSONSource(data)
	default:
		return parser.FromFile(path)
	}
}
