package prefabs

import (
	"embed"
	"path/filepath"
	"strings"
)

//go:embed *.yaml
var PrefabsFS embed.FS

func Load(name string) ([]byte, error) {
	return PrefabsFS.ReadFile(cleanPrefabPath(name))
}

func cleanPrefabPath(path string) string {
	if path == "" {
		return ""
	}
	s := filepath.ToSlash(path)
	if strings.HasPrefix(s, "prefabs/") {
		return strings.TrimPrefix(s, "prefabs/")
	}
	return s
}
