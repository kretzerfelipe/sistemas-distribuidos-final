package contracts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"futhub/notification/internal/messaging"
)

const eventSchemaSuffix = ".schema.json"

var candidateDirectories = []string{"contracts", "../../contracts", "../contracts"}

type Validator struct {
	schemasByType map[string]*jsonschema.Schema
}

func NewValidator(directory string) (*Validator, error) {
	resolved, err := resolveDirectory(directory)
	if err != nil {
		return nil, err
	}

	files, err := filepath.Glob(filepath.Join(resolved, "events", "*"+eventSchemaSuffix))
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no event schemas found in %s", resolved)
	}

	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()

	schemas := make(map[string]*jsonschema.Schema, len(files))
	for _, file := range files {
		schema, err := compiler.Compile(file)
		if err != nil {
			return nil, fmt.Errorf("compile contract %s: %w", file, err)
		}
		messageType := strings.TrimSuffix(filepath.Base(file), eventSchemaSuffix)
		schemas[messageType] = schema
	}
	return &Validator{schemasByType: schemas}, nil
}

func (v *Validator) Validate(body []byte) error {
	var header struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &header); err != nil {
		return fmt.Errorf("%w: body is not a JSON object: %v", messaging.ErrInvalidMessage, err)
	}

	schema, known := v.schemasByType[header.Type]
	if !known {
		return fmt.Errorf("%w: unsupported message type %q", messaging.ErrInvalidMessage, header.Type)
	}

	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%w: %v", messaging.ErrInvalidMessage, err)
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("%w: %v", messaging.ErrInvalidMessage, err)
	}
	return nil
}

func resolveDirectory(configured string) (string, error) {
	candidates := candidateDirectories
	if configured != "" {
		candidates = []string{configured}
	}
	for _, candidate := range candidates {
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(filepath.Join(absolute, "events")); err == nil && info.IsDir() {
			return absolute, nil
		}
	}
	return "", fmt.Errorf("contracts directory not found (set CONTRACTS_DIR); tried %v", candidates)
}
