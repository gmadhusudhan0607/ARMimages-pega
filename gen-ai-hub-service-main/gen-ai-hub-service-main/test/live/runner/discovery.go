/*
 Copyright (c) 2025 Pegasystems Inc.
 All rights reserved.
*/

package runner

import (
	"os"
	"path/filepath"
	"sort"
)

// discoverDirs returns sorted absolute paths of immediate subdirectories.
func discoverDirs(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	sort.Strings(dirs)
	return dirs
}

// filterByName returns only dirs whose base name matches the given name.
func filterByName(dirs []string, name string) []string {
	var result []string
	for _, d := range dirs {
		if filepath.Base(d) == name {
			result = append(result, d)
		}
	}
	return result
}

// dirNames returns the base names of the given directory paths.
func dirNames(dirs []string) []string {
	names := make([]string, len(dirs))
	for i, d := range dirs {
		names[i] = filepath.Base(d)
	}
	return names
}
