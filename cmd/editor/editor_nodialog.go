//go:build !dialog
// +build !dialog

package main

import "errors"

// openBackgroundDialog is a stub used when the native dialog build tag isn't set.
func openBackgroundDialog() (string, error) {
	return "", errors.New("native file dialog unavailable; build with -tags dialog to enable")
}
