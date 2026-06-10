//go:build chaos_off

// Package aws (chaos_off build): AppendChaosMiddleware registers nothing, so the
// SDK stack runs unmodified and the chaos path adds zero allocations.
package aws

import (
	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/ag4r/chaotic/engine"
)

// Step and MiddlewareOptions mirror the chaos-on surface so callers compile
// unchanged under -tags chaos_off.
type Step int

const (
	StepFinalize Step = iota
	StepBuild
)

type MiddlewareOptions struct {
	Step Step
}

func AppendChaosMiddleware(_ *awssdk.Config, _ *engine.Engine) {}

func AppendChaosMiddlewareWith(_ *awssdk.Config, _ *engine.Engine, _ MiddlewareOptions) {}
