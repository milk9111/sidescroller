package prefabs

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

func diskPrefabPath(clean string) string {
	return filepath.Join("prefabs", filepath.FromSlash(clean))
}
