//go:build chaos_off

// Package io (chaos_off build): the wrappers add no behavior and no allocation.
package io

import (
	"io"

	"github.com/RomanAgaltsev/chaotic/engine"
)

func WrapReader(r io.Reader, _ *engine.Engine) io.Reader { return r }

func WrapWriter(w io.Writer, _ *engine.Engine) io.Writer { return w }
