package main

import (
	"os"
	"path/filepath"
)

// AssetInfo holds information about an asset file.
type AssetInfo struct {
	Name string
	Path string
}

// ListImageAssets scans the assets/ folder for PNG files.
func ListImageAssets(dir string) ([]AssetInfo, error) {
	var assets []AssetInfo
	// dir := "assets"
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(info.Name())
		if ext == ".png" {
			assets = append(assets, AssetInfo{
				Name: info.Name(),
				Path: path,
			})
		}
		return nil
	})
	return assets, err
}
