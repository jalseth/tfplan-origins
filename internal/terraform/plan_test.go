package terraform

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	tfjson "github.com/hashicorp/terraform-json"
)

func TestMergeLocationsIntoPlan(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc    string
		locs    Locations
		plan    tfjson.Plan // Use the upstream struct to ensure we don't have typos in field names.
		want    map[string]any
		wantErr bool
	}{
		{
			desc:    "missing locations",
			wantErr: true,
		},
		{
			desc:    "empty resource_changes is an invalid plan",
			plan:    tfjson.Plan{ResourceChanges: []*tfjson.ResourceChange{}},
			wantErr: true,
		},
		{
			desc: "odd numbered resource address is invalid",
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{Address: "module.name.res_type"},
				},
			},
			wantErr: true,
		},
		{
			desc: "simple one-to-one",
			locs: Locations{
				"foo.bar":   {File: "foo.tf", Line: 123},
				"foo.bar#a": {File: "foo.tf", Line: 124},
			},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address: "foo.bar",
						Change: &tfjson.Change{
							After: map[string]any{
								"a": 1,
								"b": 2,
							},
						},
					},
				},
			},
			want: map[string]any{
				resourceChangesField: []map[string]any{
					{
						addressField: "foo.bar",
						"change": map[string]any{
							"after": map[string]any{
								"a": float64(1),
								"b": float64(2),
							},
							"before": nil,
						},
						locationField: &Location{
							File: "foo.tf",
							Line: 123,
						},
						fieldLocField: Locations{
							"foo.bar#a": &Location{File: "foo.tf", Line: 124},
						},
					},
				},
			},
		},
		{
			desc: "complex",
			locs: Locations{
				"foo.bar": {
					File: "foo.tf",
					Line: 123,
				},
				"module.baz.qwerty.123456": {
					File: "modules/baz/qwerty.tf",
					Line: 999,
				},
				"module.nested.module.git_source": {
					File: "modules/nested/main.tf",
					Line: 10,
				},
			},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{Address: "foo.bar"},
					{Address: "not.found"},
					{Address: "module.baz.qwerty.123456"},
					{Address: "module.nested.module.git_source.res_type.res_name"},
				},
			},
			want: map[string]any{
				resourceChangesField: []map[string]any{
					{
						addressField: "foo.bar",
						locationField: &Location{
							File: "foo.tf",
							Line: 123,
						},
					},
					{
						addressField: "not.found",
					},
					{
						addressField: "module.baz.qwerty.123456",
						locationField: &Location{
							File: "modules/baz/qwerty.tf",
							Line: 999,
						},
					},
					{
						addressField: "module.nested.module.git_source.res_type.res_name",
						locationField: &Location{
							File: "modules/nested/main.tf",
							Line: 10,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			plan := mustConvertToMap(t, tc.plan)
			err := MergeLocationsIntoPlan(tc.locs, plan)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("MergeLocationsIntoPlan() error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if diff := cmp.Diff(plan, tc.want); diff != "" {
				t.Errorf("MergeLocationsIntoPlan() produced an unexpected diff:\n%s", diff)
			}
		})
	}
}

func mustConvertToMap(t *testing.T, v any) map[string]any {
	t.Helper()

	by, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	m := make(map[string]any)
	if err := json.Unmarshal(by, &m); err != nil {
		t.Fatal(err)
	}
	return m
}
