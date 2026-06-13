package engine

import (
	"testing"
	"time"

	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestSeverityString(t *testing.T) {
	cases := []struct {
		sev  Severity
		want string
	}{
		{SeverityInfo, "info"},
		{SeverityWarn, "warn"},
		{SeverityHigh, "high"},
		{Severity(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.sev.String(); got != tc.want {
			t.Errorf("Severity(%d).String() = %q, want %q", tc.sev, got, tc.want)
		}
	}
}

func TestTerminalKindName(t *testing.T) {
	cases := []struct {
		kind fault.Kind
		want string
	}{
		{fault.KindPanic, "panic"},
		{fault.KindConnDrop, "conn_drop"},
		{fault.KindDisconnect, "disconnect"},
		{fault.KindLatency, ""}, // non-terminal => empty
		{fault.KindUnknown, ""}, // unknown => empty
	}
	for _, tc := range cases {
		if got := terminalKindName(tc.kind); got != tc.want {
			t.Errorf("terminalKindName(%v) = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestSpecLatency(t *testing.T) {
	cases := []struct {
		name    string
		fs      FaultSpec
		wantDur time.Duration
		wantOK  bool
	}{
		{"latency parses", FaultSpec{Type: "latency", Duration: "250ms"}, 250 * time.Millisecond, true},
		{"latency unparseable", FaultSpec{Type: "latency", Duration: "nope"}, 0, false},
		{"jittered uses max", FaultSpec{Type: "jittered", Max: "2s"}, 2 * time.Second, true},
		{"jittered unparseable", FaultSpec{Type: "jittered", Max: "??"}, 0, false},
		{"non-latency fault", FaultSpec{Type: "error", Message: "boom"}, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, ok := specLatency(tc.fs)
			if ok != tc.wantOK || d != tc.wantDur {
				t.Fatalf("specLatency(%+v) = (%v, %v), want (%v, %v)", tc.fs, d, ok, tc.wantDur, tc.wantOK)
			}
		})
	}
}

func TestLintName(t *testing.T) {
	if got := lintName(""); got != "<unnamed>" {
		t.Errorf("lintName(\"\") = %q, want %q", got, "<unnamed>")
	}
	if got := lintName("flaky"); got != "flaky" {
		t.Errorf("lintName(%q) = %q, want %q", "flaky", got, "flaky")
	}
}
