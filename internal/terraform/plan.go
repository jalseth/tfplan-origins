package terraform

import (
	"fmt"
	"strings"
)

const (
	resourceChangesField = "resource_changes"
	addressField         = "address"
	locationField        = "_loc"
)

func MergeLocationsIntoPlan(locs Locations, plan map[string]any) error {
	if len(locs) == 0 {
		return fmt.Errorf("locations must be supplied")
	}
	if plan == nil || plan[resourceChangesField] == nil {
		return fmt.Errorf("plan with %s must be supplied", resourceChangesField)
	}
	resourceChanges, ok := plan[resourceChangesField].([]any)
	if !ok {
		return fmt.Errorf("%s: not a list", resourceChangesField)
	}

	rcs := make([]map[string]any, 0, len(resourceChanges))
	for i, resourceChange := range resourceChanges {
		rc, ok := resourceChange.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: %d: not an object", resourceChangesField, i)
		}

		address, ok := rc[addressField]
		if !ok {
			return fmt.Errorf("%s: %d: missing %q", resourceChangesField, i, addressField)
		}
		addr, ok := address.(string)
		if !ok {
			return fmt.Errorf("%s: %d: address is not a string: %v", resourceChangesField, i, address)
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
		rcs = append(rcs, rc)
	}

	plan[resourceChangesField] = rcs

	return nil
}
