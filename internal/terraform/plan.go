package terraform

import (
	"fmt"
	"strings"
)

const (
	resourceChangesField = "resource_changes"
	addressField         = "address"
	locationField        = "_loc"
	fieldLocField        = "_field_loc"
)

func MergeLocationsIntoPlan(locs Locations, plan map[string]any) error {
	if len(locs) == 0 {
		return fmt.Errorf("locations must be supplied")
	}
	if plan == nil {
		return fmt.Errorf("plan must be supplied")
	}
	resourceChanges, ok := lookup[[]any](plan, resourceChangesField)
	if !ok {
		return fmt.Errorf("plan is missing required field: %s", resourceChangesField)
	}

	rcs := make([]map[string]any, 0, len(resourceChanges))
	for i, resourceChange := range resourceChanges {
		rc, ok := resourceChange.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: %d: not an object", resourceChangesField, i)
		}
		addr, ok := lookup[string](rc, "address")
		if !ok {
			return fmt.Errorf("%s: %d: address must be present", resourceChangesField, i)
		}

		// We may have a partial match for modules where the source is not available.
		// Walk up the address until we find a match.
		sp := strings.Split(addr, ".")
		if len(sp)%2 != 0 {
			return fmt.Errorf("%s: %d: resource address is invalid: %s", resourceChangesField, i, addr)
		}
		for i := 0; i < len(sp); i += 2 {
			addr = strings.Join(sp[0:len(sp)-i], ".")
			if loc, ok := locs[addr]; ok {
				rc[locationField] = loc
				break
			}
		}
		if fl := fieldLocations(locs, rc, addr); fl != nil {
			rc[fieldLocField] = fl
		}
		rcs = append(rcs, rc)
	}

	plan[resourceChangesField] = rcs

	return nil
}

func fieldLocations(locs Locations, rc map[string]any, addr string) Locations {
	change, ok := lookup[map[string]any](rc, "change")
	if !ok {
		return nil
	}
	after, ok := lookup[map[string]any](change, "after")
	if !ok {
		return nil
	}

	fl := make(Locations)
	for field := range after {
		fieldAddr := addr + "#" + field
		if loc, ok := locs[fieldAddr]; ok {
			fl[fieldAddr] = loc
		}
	}
	return fl
}

func lookup[T any](m map[string]any, k string) (value T, ok bool) {
	v, ok := m[k]
	value, ok = v.(T)
	return
}
