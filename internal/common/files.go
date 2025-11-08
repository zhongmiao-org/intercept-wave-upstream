package common

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AssetsDir returns the directory for static JSON assets.
// Overridable via env ASSETS_DIR; defaults to "assets".
func AssetsDir() string {
	if v := os.Getenv("ASSETS_DIR"); v != "" {
		return v
	}
	return "assets"
}

// JoinAssets builds a path under the assets directory.
func JoinAssets(parts ...string) string {
	base := AssetsDir()
	p := filepath.Join(parts...)
	return filepath.Join(base, p)
}

// LoadJSONDynamic loads a JSON file into an untyped Go value (map/array/etc).
func LoadJSONDynamic(path string) (any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return v, nil
}
