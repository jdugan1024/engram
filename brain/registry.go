// ABOUTME: Schema registry — loads and indexes JSON Schema files embedded in the binary.
// ABOUTME: Schemas live in brain/schemas/ as <record_type>@<version>.json files.

package brain

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed schemas/*.json
var schemaFS embed.FS

// SchemaEntry holds a parsed JSON Schema alongside its raw bytes.
type SchemaEntry struct {
	RecordType    string
	SchemaVersion string
	Raw           []byte
	Schema        map[string]any
}

// schemaRegistry is the process-wide registry populated at init time.
var schemaRegistry map[string]*SchemaEntry

func init() {
	if err := loadSchemas(); err != nil {
		panic("schema registry: " + err.Error())
	}
}

// loadSchemas reads all *.json files from the embedded schemas/ directory and
// indexes them by "<record_type>@<version>" key.
func loadSchemas() error {
	schemaRegistry = make(map[string]*SchemaEntry)

	entries, err := schemaFS.ReadDir("schemas")
	if err != nil {
		return fmt.Errorf("read schemas dir: %w", err)
	}

	for _, de := range entries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".json") {
			continue
		}

		// Filename format: <record_type>@<version>.json
		// e.g. crm.contact@1.0.0.json → record_type="crm.contact", version="1.0.0"
		base := strings.TrimSuffix(de.Name(), ".json")
		atIdx := strings.LastIndex(base, "@")
		if atIdx < 0 {
			return fmt.Errorf("schema file %q: expected <record_type>@<version>.json format", de.Name())
		}
		recordType := base[:atIdx]
		version := base[atIdx+1:]

		raw, err := schemaFS.ReadFile("schemas/" + de.Name())
		if err != nil {
			return fmt.Errorf("read schema %q: %w", de.Name(), err)
		}

		var parsed map[string]any
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return fmt.Errorf("parse schema %q: %w", de.Name(), err)
		}

		key := recordType + "@" + version
		schemaRegistry[key] = &SchemaEntry{
			RecordType:    recordType,
			SchemaVersion: version,
			Raw:           raw,
			Schema:        parsed,
		}
	}

	if len(schemaRegistry) == 0 {
		return fmt.Errorf("no schemas loaded from embedded schemas/ directory")
	}
	return nil
}

// SchemaFor returns the SchemaEntry for the given record type and version.
// Returns an error if no matching schema is registered.
func SchemaFor(recordType, version string) (*SchemaEntry, error) {
	key := recordType + "@" + version
	se, ok := schemaRegistry[key]
	if !ok {
		return nil, fmt.Errorf("no schema registered for %q at version %q", recordType, version)
	}
	return se, nil
}

// KnownRecordTypes returns the list of all registered record types (deduplicated).
func KnownRecordTypes() []string {
	seen := make(map[string]bool)
	var types []string
	for _, se := range schemaRegistry {
		if !seen[se.RecordType] {
			seen[se.RecordType] = true
			types = append(types, se.RecordType)
		}
	}
	return types
}
