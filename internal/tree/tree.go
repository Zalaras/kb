package tree

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Walk lists every regular file under root as a sorted, forward-slash, root-relative
// path. Symlinks are not followed; skip (slash-relative directories never entered) and
// keepHidden (the dot-directories it does enter) stand in for git ls-files. Any directory whose name contains node_modules is skipped too.
func Walk(root string, skip, keepHidden []string) ([]string, error) {
	skipDirs, keptHiddenDirs := map[string]bool{}, map[string]bool{}
	for _, d := range skip {
		skipDirs[strings.TrimSuffix(d, "/")] = true
	}
	for _, d := range keepHidden {
		keptHiddenDirs[d] = true
	}
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if skipDirs[rel] || strings.Contains(base, "node_modules") ||
				(strings.HasPrefix(base, ".") && !keptHiddenDirs[base]) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() {
			out = append(out, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", root, err)
	}
	sort.Strings(out)
	return out, nil
}
