/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */
package testutils

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root of the filesystem
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func TestNoTestutilImportInNonTestFiles(t *testing.T) {
	root, err := getProjectRoot()
	assert.NoError(t, err)

	// Test packages that should only be imported by test files
	testOnlyPackages := []string{"testutils", "cntxtest"}

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip the "build" directory
			if info.Name() == "build" || info.Name() == "test" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip test files and compiled files
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".gen.go") {
			return nil
		}

		// Check only Go files
		if strings.HasSuffix(path, ".go") {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				// Log the error and continue
				t.Logf("Error parsing file %s: %v", path, err)
			}

			for _, imp := range node.Imports {
				for _, testPkg := range testOnlyPackages {
					if strings.Contains(imp.Path.Value, testPkg) {
						t.Errorf("Non-test file %s imports test-only package %s", path, testPkg)
					}
				}
			}
		}
		return nil
	})

	assert.NoError(t, err)
}
