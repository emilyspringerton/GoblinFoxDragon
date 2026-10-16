package main

import "testing"

func TestReconcileFlow(t *testing.T) {
	cases := []struct {
		name                  string
		local, synced, remote int
		want                  int
		changed               bool
	}{
		{"in sync", 100, 100, 100, 100, false},
		{"hat sale credited remotely", 100, 100, 188, 188, true},
		{"hat bought on web", 500, 500, 250, 250, true},
		{"unsynced local earnings are never swallowed", 130, 100, 50, 130, false},
		{"unsynced local spend is never overwritten", 70, 100, 100, 70, false},
	}
	for _, c := range cases {
		got, ch := reconcileFlow(c.local, c.synced, c.remote)
		if got != c.want || ch != c.changed {
			t.Errorf("%s: got (%d,%v) want (%d,%v)", c.name, got, ch, c.want, c.changed)
		}
	}
}
