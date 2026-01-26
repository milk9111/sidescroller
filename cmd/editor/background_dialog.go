//go:build dialog
// +build dialog

package main

import (
	"github.com/sqweek/dialog"
)

// openBackgroundDialog opens the native file dialog and returns the selected path.
func openBackgroundDialog() (string, error) {
	return dialog.File().Filter("Image files", "png", "jpg", "jpeg").Title("Select background image").Load()
}
