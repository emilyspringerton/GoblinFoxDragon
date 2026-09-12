package main

import "testing"

// TestCloneInventory_IsIndependentCopy -- S252-01's real correctness requirement: the snapshot
// taken for delta-sync comparison must not alias the live map, or a later mutation of one would
// silently corrupt the other.
func TestCloneInventory_IsIndependentCopy(t *testing.T) {
	orig := map[string]int{"earth-crystal": 3}
	clone := cloneInventory(orig)
	clone["earth-crystal"] = 99
	clone["worm-sinew"] = 1

	if orig["earth-crystal"] != 3 {
		t.Errorf("mutating the clone affected the original: %+v", orig)
	}
	if _, ok := orig["worm-sinew"]; ok {
		t.Errorf("clone's new key leaked into the original: %+v", orig)
	}
}

func TestInventoryEqual(t *testing.T) {
	cases := []struct {
		name string
		a, b map[string]int
		want bool
	}{
		{"both empty", map[string]int{}, map[string]int{}, true},
		{"identical", map[string]int{"a": 1, "b": 2}, map[string]int{"a": 1, "b": 2}, true},
		{"different value", map[string]int{"a": 1}, map[string]int{"a": 2}, false},
		{"different key set, same len", map[string]int{"a": 1}, map[string]int{"b": 1}, false},
		{"different length", map[string]int{"a": 1}, map[string]int{"a": 1, "b": 2}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := inventoryEqual(c.a, c.b); got != c.want {
				t.Errorf("inventoryEqual(%+v, %+v) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}
