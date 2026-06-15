# Activate chaos from a one-line terms string

This example shows the `source/terms` DSL: a single line activates chaos with no
Go rule-building code.

## What it shows

- `terms.Compile("checkout: kind(explicit),name(checkout)=2*error(\"payment down\")")`
  parses one line into rules.
- The rules are added to the engine and drive an explicit `chaos.Point`.
- The first two checkout points fail; the third passes (`2*` = `Times(2)`); the
  rule's hit counter proves it fired exactly twice.

This is the same text form that `source/env` and the `source/http` admin endpoint
use, so the one-liner you test with is the one you can ship to an operator.

## Run it

go test ./...
