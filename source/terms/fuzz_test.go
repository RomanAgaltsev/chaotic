package terms

import "testing"

func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		``,
		`error("boom")`,
		`flaky: kind(http_client),name(/u/*)=2*latency(200ms)`,
		`a; b; c`,
		`attr(k=v)=conndrop`,
		`40%panic("x")`,
		`->`,
		`kind(`,
		`(((`,
		`name()=off`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		// Must never panic; an error return is fine.
		_, _ = Parse(s)
	})
}
