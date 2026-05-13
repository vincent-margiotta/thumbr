package notes

import "testing"

func TestDeriveContinuation(t *testing.T) {
	cases := []struct {
		stem string
		want string
		ok   bool
	}{
		{"1", "1a", true},
		{"16", "16a", true},
		{"16a", "16a1", true},
		{"16a1", "16a1a", true},
		{"16a1a", "16a1a1", true},
		{"16a9", "16a9a", true},
		{"a", "a1", true},
		{"z", "z1", true},
		{"", "", false},
		{"foo-bar", "", false},
		{"16_a", "", false},
	}
	for _, tc := range cases {
		got, ok := DeriveContinuation(tc.stem)
		if ok != tc.ok || got != tc.want {
			t.Errorf("DeriveContinuation(%q) = (%q, %v), want (%q, %v)", tc.stem, got, ok, tc.want, tc.ok)
		}
	}
}

func TestDeriveBranch(t *testing.T) {
	cases := []struct {
		stem string
		want string
		ok   bool
	}{
		{"1", "2", true},
		{"16", "17", true},
		{"16a", "16b", true},
		{"16a1", "16a2", true},
		{"16a9", "16a10", true},
		{"16a1a", "16a1b", true},
		{"16z", "", false}, // overflow
		{"16Z", "", false}, // overflow
		{"a", "b", true},
		{"z", "", false},
		{"Z", "", false},
		{"", "", false},
		{"foo-bar", "", false},
		// multi-letter last component: continue works but branch does not
		{"16ab", "", false},
	}
	for _, tc := range cases {
		got, ok := DeriveBranch(tc.stem)
		if ok != tc.ok || got != tc.want {
			t.Errorf("DeriveBranch(%q) = (%q, %v), want (%q, %v)", tc.stem, got, ok, tc.want, tc.ok)
		}
	}
}
