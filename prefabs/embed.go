package prefabs

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed scripts/*.tengo
var ScriptsFS embed.FS

func LoadScript(name string) ([]byte, error) {
	clean := cleanScriptPath(name)
	return ScriptsFS.ReadFile(clean)
}

//go:embed *.yaml
var PrefabsFS embed.FS

func Load(name string) ([]byte, error) {
	clean := cleanPrefabPath(name)
	if data, err := os.ReadFile(diskPrefabPath(clean)); err == nil {
		return data, nil
	}
	return PrefabsFS.ReadFile(clean)
}

func ModTime(name string) (time.Time, bool) {
	clean := cleanPrefabPath(name)
	info, err := os.Stat(diskPrefabPath(clean))
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
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

func cleanScriptPath(path string) string {
	if path == "" {
		return ""
	}

	s := filepath.ToSlash(path)

	if after, ok := strings.CutPrefix(s, "prefabs/scripts/"); ok {
		s = after
	}

	if after, ok := strings.CutPrefix(s, "prefabs/"); ok {
		s = after
	}

	if after, ok := strings.CutPrefix(s, "scripts/"); ok {
		s = after
	}

	return fmt.Sprintf("scripts/%s", s)
}

func diskPrefabPath(clean string) string {
	return filepath.Join("prefabs", filepath.FromSlash(clean))
}
