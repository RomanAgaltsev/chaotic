//go:build chaos_off

package aws

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/ag4r/chaotic/engine"
)

func TestZeroAllocUnderChaosOff(t *testing.T) {
	eng := engine.New()
	avg := testing.AllocsPerRun(100, func() {
		var cfg awssdk.Config
		AppendChaosMiddleware(&cfg, eng)
	})
	if avg != 0 {
		t.Fatalf("allocs/op = %v, want 0 under chaos_off", avg)
	}
}
