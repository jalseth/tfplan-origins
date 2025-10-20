package terraform

import (
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// Locations is a map from a Terraform resource address to a source location.
// Resources within modules and submodules are supported.
//
// Address formats:
// - `<resource_type>.<resource_name>`
// - `<resource_type>.<resource_name>#<field_name>`
// - `module.<module_name>`
// - `module.<module_name>.<resource_type>.<resource_name>`
// - `module.<module_name>.<resource_type>.<resource_name>#<field_name>`
type Locations map[string]*Location

// Location represents a specific location in a file.
type Location struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// ParseLocations accepts a fs.FS and directory pointing to the root dir
// Terraform config directory within that filesystem.
func ParseLocations(fs fs.FS, rootDir string) (Locations, error) {
	return parseDir(fs, rootDir, "")
}

// parseDir recursively parses all resource and module source locations from the
// specified directory.
func parseDir(fs fs.FS, dir string, modAddr string) (Locations, error) {
	mod, err := parseModuleDir(fs, dir)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", dir, err)
	}

	resourceLocations := make(Locations)
	for name, file := range mod {
		rl, err := parseBlocks(fs, file, name, dir, modAddr)
		if err != nil {
			return nil, fmt.Errorf("parse modules for %s: %w", filepath.Join(dir, name), err)
		}
		maps.Copy(resourceLocations, rl)
	}

	return resourceLocations, nil
}

func parseBlocks(fs fs.FS, file *hcl.File, fileName, dir, modAddr string) (Locations, error) {
	content, _, diag := file.Body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			// resource "resource_type" "resource_name" { ... }
			{Type: "resource", LabelNames: []string{"type", "name"}},
			// module "name" { source = "", ... }
			{Type: "module", LabelNames: []string{"name"}},
		},
	})
	if diag.HasErrors() {
		return nil, diag
	}

	fp := filepath.Join(dir, fileName)
	rl := make(Locations)

	var baseAddr string
	if modAddr != "" {
		baseAddr = modAddr + "."
	}

	ctx := &hcl.EvalContext{}
	for _, block := range content.Blocks {
		attrs, diag := block.Body.JustAttributes()
		if diag.HasErrors() {
			return nil, diag
		}

		var resAddr string
		switch block.Type {
		case "resource":
			resAddr = baseAddr + block.Labels[0] + "." + block.Labels[1]
			for _, attr := range attrs {
				fieldAddr := resAddr + "#" + attr.Name
				rl[fieldAddr] = &Location{File: fp, Line: attr.Range.Start.Line}
			}

		case "module":
			modName := block.Labels[0]
			resAddr = baseAddr + "module." + modName
			source, ok := attrs["source"]
			if !ok {
				return nil, fmt.Errorf("%s: module %q is missing %q attribute", fp, modName, "source")
			}
			srcExpr, diag := source.Expr.Value(ctx)
			if diag.HasErrors() {
				return nil, diag
			}
			src := srcExpr.AsString()

			if !strings.HasPrefix(src, "./") && !strings.HasPrefix(src, "../") {
				// If the source of the module is not in the fs, such as with Git repo sources,
				// add a globbing wildcard to indicate this.
				rl[resAddr+".*"] = &Location{File: fp, Line: block.DefRange.Start.Line}

			} else {
				// Otherwise, continue to parse submodule directories until not more are found.
				submodRL, err := parseDir(fs, filepath.Join(dir, src), resAddr)
				if err != nil {
					return nil, fmt.Errorf("%s: module %s: parse submodule %s: %w", fp, modAddr, modName, err)
				}
				maps.Copy(rl, submodRL)
			}
		}

		// Finally, add the resource or module to the locations map.
		rl[resAddr] = &Location{File: fp, Line: block.DefRange.Start.Line}
	}

	return rl, nil
}

func parseModuleDir(files fs.FS, dir string) (map[string]*hcl.File, error) {
	fds, err := fs.ReadDir(files, dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	mod := make(map[string]*hcl.File)
	parser := hclparse.NewParser()
	for _, fd := range fds {
		if fd.IsDir() {
			continue
		}
		if filepath.Ext(fd.Name()) != ".tf" {
			continue
		}

		fp := filepath.Join(dir, fd.Name())
		contents, err := fs.ReadFile(files, fp)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", fp, err)
		}
		parsed, diag := parser.ParseHCL(contents, fp)
		if diag.HasErrors() {
			return nil, fmt.Errorf("parse %s: %w", fp, diag)
		}
		mod[fd.Name()] = parsed
	}

	return mod, nil
}
