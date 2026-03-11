package editorio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/milk9111/sidescroller/cmd/editor/model"
	"github.com/milk9111/sidescroller/levels"
)

type AssetInfo struct {
	Name     string
	DiskPath string
	Relative string
}

func NormalizeLevelTarget(target string) string {
	base := filepath.Base(strings.TrimSpace(target))
	if base == "." || base == string(filepath.Separator) || base == "" {
		base = "untitled"
	}
	if filepath.Ext(base) == "" {
		base += ".json"
	}
	return base
}

func ResolveLevelPath(workspaceRoot, levelDir, target string) string {
	return filepath.Join(workspaceRoot, levelDir, NormalizeLevelTarget(target))
}

func LoadLevel(workspaceRoot, levelDir, target string) (*model.LevelDocument, string, error) {
	path := ResolveLevelPath(workspaceRoot, levelDir, target)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read level %q: %w", path, err)
	}
	var level levels.Level
	if err := json.Unmarshal(data, &level); err != nil {
		return nil, "", fmt.Errorf("decode level %q: %w", path, err)
	}
	return model.FromRuntimeLevel(&level), NormalizeLevelTarget(target), nil
}

func SaveLevel(workspaceRoot, levelDir, target string, doc *model.LevelDocument) (string, error) {
	normalized := NormalizeLevelTarget(target)
	path := ResolveLevelPath(workspaceRoot, levelDir, normalized)
	runtimeLevel := doc.ToRuntimeLevel()
	data, err := json.MarshalIndent(runtimeLevel, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode level %q: %w", normalized, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write level %q: %w", path, err)
	}
	return normalized, nil
}

func ScanPNGAssets(workspaceRoot, assetDir string) ([]AssetInfo, error) {
	root := filepath.Join(workspaceRoot, assetDir)
	assets := make([]AssetInfo, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(entry.Name())) != ".png" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		assets = append(assets, AssetInfo{
			Name:     entry.Name(),
			DiskPath: path,
			Relative: filepath.ToSlash(rel),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(assets, func(i, j int) bool {
		if assets[i].Name == assets[j].Name {
			return assets[i].Relative < assets[j].Relative
		}
		return assets[i].Name < assets[j].Name
	})
	return assets, nil
}

func PromptForLevelSize(defaultWidth, defaultHeight int) (int, int, error) {
	reader := bufio.NewReader(os.Stdin)
	width, err := promptDimension(reader, "Level width", defaultWidth)
	if err != nil {
		return 0, 0, err
	}
	height, err := promptDimension(reader, "Level height", defaultHeight)
	if err != nil {
		return 0, 0, err
	}
	return width, height, nil
}

func promptDimension(reader *bufio.Reader, label string, fallback int) (int, error) {
	fmt.Printf("%s [%d]: ", label, fallback)
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return 0, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return fallback, nil
	}
	var value int
	if _, scanErr := fmt.Sscanf(line, "%d", &value); scanErr != nil || value <= 0 {
		return fallback, nil
	}
	return value, nil
}
