package terraform

import (
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
)

func TestParseLocations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc    string
		files   map[string]string
		rootDir string
		want    Locations
		wantErr bool
	}{
		{
			desc:    "invalid path",
			rootDir: "/non-exist-9834542",
			wantErr: true,
		},
		{
			desc:    "no files",
			rootDir: ".",
			want:    make(Locations),
		},
		{
			desc:    "no resource or module blocks",
			rootDir: ".",
			files: map[string]string{
				"invalid.tf": `terraform {
}`,
			},
			want: make(Locations),
		},
		{
			desc:    "submodule path does not exist",
			rootDir: ".",
			files: map[string]string{
				"main.tf": `module "foo" {
  source = "./does/not/exist"
}`,
			},
			wantErr: true,
		},
		{
			desc:    "simple: ignoring unknown file types and block types",
			rootDir: ".",
			files: map[string]string{
				"valid.tf": `resource "local_file" "foo" {
  filename = "foo.txt"
  content = "hi"
}

some_other_thing {
  abc = 123
}`,
				"not_terraform.json": `{"foo": "bar"}`,
			},
			want: Locations{
				"local_file.foo":          &Location{File: "valid.tf", Line: 1},
				"local_file.foo#filename": &Location{File: "valid.tf", Line: 2},
				"local_file.foo#content":  &Location{File: "valid.tf", Line: 3},
			},
		},
		{
			desc:    "simple: single module dependency in subdir",
			rootDir: ".",
			files: map[string]string{
				"main.tf": `module "foo" {
  source = "./cluster"
}`,
				"cluster/main.tf": `
resource "type_a" "res_a" {
  field_a = "a"
}
resource "type_a" "res_b" {
  field_a = "b"
}
resource "type_b" "res_c" {
  field_abc = "${type_a.res_a.field_a}.${type_a.res_b.field_a}"
}`,
			},
			want: Locations{
				"module.foo":                        &Location{File: "main.tf", Line: 1},
				"module.foo.type_a.res_a":           &Location{File: "cluster/main.tf", Line: 2},
				"module.foo.type_a.res_a#field_a":   &Location{File: "cluster/main.tf", Line: 3},
				"module.foo.type_a.res_b":           &Location{File: "cluster/main.tf", Line: 5},
				"module.foo.type_a.res_b#field_a":   &Location{File: "cluster/main.tf", Line: 6},
				"module.foo.type_b.res_c":           &Location{File: "cluster/main.tf", Line: 8},
				"module.foo.type_b.res_c#field_abc": &Location{File: "cluster/main.tf", Line: 9},
			},
		},
		{
			desc:    "simple: single module in parent relative dir",
			rootDir: "env",
			files: map[string]string{
				"env/main.tf": `module "foo" {
  source = "../modules/cluster"
}`,
				"modules/cluster/main.tf": `
resource "type_a" "res_a" {
  field_a = "a"
}`,
			},
			want: Locations{
				"module.foo":                      &Location{File: "env/main.tf", Line: 1},
				"module.foo.type_a.res_a":         &Location{File: "modules/cluster/main.tf", Line: 2},
				"module.foo.type_a.res_a#field_a": &Location{File: "modules/cluster/main.tf", Line: 3},
			},
		},
		{
			desc:    "complex: multiple modules",
			rootDir: ".",
			files: map[string]string{
				"main.tf": `resource "foo" "bar" {
  baz = {}
}

module "single" {
  source = "./single"
}

module "double" {
  source = "./double"
}`,
				"single/main.tf": `resource "mod_single" "res_123" {}`,
				"double/main.tf": `resource "mod_double" "res_789" {}

module "triple" {
  source = "../modules/triple"
}				`,
				"modules/triple/some_file.tf": `data "ignore" "this" {
  abc = "xyz"
}

# HCL comment
resource "final" "final" {
  // Another comment
  v = [1, 2, 3]
}`,
			},
			want: Locations{
				"foo.bar":                                   &Location{File: "main.tf", Line: 1},
				"foo.bar#baz":                               &Location{File: "main.tf", Line: 2},
				"module.single":                             &Location{File: "main.tf", Line: 5},
				"module.double":                             &Location{File: "main.tf", Line: 9},
				"module.single.mod_single.res_123":          &Location{File: "single/main.tf", Line: 1},
				"module.double.mod_double.res_789":          &Location{File: "double/main.tf", Line: 1},
				"module.double.module.triple":               &Location{File: "double/main.tf", Line: 3},
				"module.double.module.triple.final.final":   &Location{File: "modules/triple/some_file.tf", Line: 6},
				"module.double.module.triple.final.final#v": &Location{File: "modules/triple/some_file.tf", Line: 8},
			},
		},
		{
			desc:    "handle modules not using local filesytem",
			rootDir: ".",
			files: map[string]string{
				"main.tf": `
module "git_source" {
  source = "git+ssh://......"
}`,
			},
			want: Locations{
				"module.git_source":   &Location{File: "main.tf", Line: 2},
				"module.git_source.*": &Location{File: "main.tf", Line: 2},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			files := make(fstest.MapFS, len(tc.files))
			for name, contents := range tc.files {
				files[name] = &fstest.MapFile{Data: []byte(contents)}
			}

			got, err := ParseLocations(files, tc.rootDir)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("ParseLocations() error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("ParseLocations() produced unexpected diff:\n%s", diff)
			}
		})
	}
}
