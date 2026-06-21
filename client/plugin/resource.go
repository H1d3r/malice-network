package plugin

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func cleanResourceDir(dirname string) (string, error) {
	dirname = strings.TrimSpace(filepath.ToSlash(dirname))
	if dirname == "" {
		return "", nil
	}
	if path.IsAbs(dirname) {
		return "", fmt.Errorf("invalid resource directory: %s", dirname)
	}
	for _, part := range strings.Split(dirname, "/") {
		if part == ".." {
			return "", fmt.Errorf("invalid resource directory: %s", dirname)
		}
	}

	cleaned := path.Clean(dirname)
	if cleaned == "." {
		return "", nil
	}
	if path.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("invalid resource directory: %s", dirname)
	}

	return cleaned, nil
}

func listDirEntryNames(entries []fs.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names
}

func listFilesystemResourceDir(rootDir, dirname string) ([]string, error) {
	cleaned, err := cleanResourceDir(dirname)
	if err != nil {
		return nil, err
	}

	resourceDir := filepath.Join(rootDir, filepath.FromSlash(cleaned))
	entries, err := os.ReadDir(resourceDir)
	if err != nil {
		return nil, fmt.Errorf("list resource directory %s: %w", dirname, err)
	}
	return listDirEntryNames(entries), nil
}

func listEmbedResourceDir(plug *EmbedPlugin, dirname string) ([]string, error) {
	cleaned, err := cleanResourceDir(dirname)
	if err != nil {
		return nil, err
	}

	resourceDir := "resources"
	if cleaned != "" {
		resourceDir += "/" + cleaned
	}

	entries, err := plug.ReadDir(resourceDir)
	if err != nil {
		return nil, fmt.Errorf("list resource directory %s: %w", dirname, err)
	}
	return listDirEntryNames(entries), nil
}
