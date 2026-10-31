package clamm

import (
	"embed"
	"path"
	"sort"
	"strings"
)

//go:embed modules
var fs embed.FS

func Modules() ([][]byte, error) {
	entries, err := fs.ReadDir("modules")
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".mv") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	codes := make([][]byte, 0, len(names))
	for _, name := range names {
		bz, err := fs.ReadFile(path.Join("modules", name))
		if err != nil {
			return nil, err
		}
		codes = append(codes, bz)
	}

	return codes, nil
}
