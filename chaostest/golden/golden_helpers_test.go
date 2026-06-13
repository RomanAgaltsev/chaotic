package golden

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGoldenPath(t *testing.T) {
	want := filepath.Join("testdata", "demo.golden")
	if got := goldenPath("demo"); got != want {
		t.Errorf("goldenPath(demo) = %q, want %q", got, want)
	}
}

func TestDiffSequences(t *testing.T) {
	cases := []struct {
		name        string
		a, b        []string
		wantContain string // "" => expect no diff
	}{
		{"identical", []string{"x", "y"}, []string{"x", "y"}, ""},
		{"different length", []string{"x"}, []string{"x", "y"}, "fired count"},
		{"same length, differing element", []string{"x", "y"}, []string{"x", "z"}, "fired[1]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diff := diffSequences(tc.a, tc.b)
			if tc.wantContain == "" {
				if diff != "" {
					t.Fatalf("diffSequences = %q, want empty", diff)
				}
				return
			}
			if !strings.Contains(diff, tc.wantContain) {
				t.Fatalf("diffSequences = %q, want it to contain %q", diff, tc.wantContain)
			}
		})
	}
}

func TestUpdateEnabled(t *testing.T) {
	t.Run("env set enables", func(t *testing.T) {
		t.Setenv("CHAOS_UPDATE_GOLDEN", "1")
		if !updateEnabled() {
			t.Fatal("updateEnabled() = false with CHAOS_UPDATE_GOLDEN=1, want true")
		}
	})
	t.Run("env unset/other is disabled", func(t *testing.T) {
		t.Setenv("CHAOS_UPDATE_GOLDEN", "0")
		if updateEnabled() {
			t.Fatal("updateEnabled() = true with CHAOS_UPDATE_GOLDEN=0, want false")
		}
	})
}

func TestWriteReadGoldenRoundTrip(t *testing.T) {
	events := []goldenEvent{
		{Fired: true, Rule: "slow", Kind: 1, Name: "/users", Method: "GET", FaultKind: 2, LatencyNS: 1500},
		{Fired: false, Rule: "flaky", Kind: 1, Name: "/orders", Reason: "counter exhausted"},
	}
	path := filepath.Join(t.TempDir(), "nested", "rt.golden")

	if err := writeGolden(path, events); err != nil {
		t.Fatalf("writeGolden: %v", err)
	}
	got, err := readGolden(path)
	if err != nil {
		t.Fatalf("readGolden: %v", err)
	}
	if !reflect.DeepEqual(got, events) {
		t.Fatalf("round trip mismatch:\n got = %+v\nwant = %+v", got, events)
	}
}
