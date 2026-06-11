# chaotic-points

Static analyzer for chaotic explicit injection points. Discovers every
`chaos.Point` / `chaos.PointWith` call site in a module and gates a rules config
against typo'd point names, so a rule that targets a nonexistent point fails CI.

## Install

go install github.com/ag4r/chaotic/cmd/chaotic-points@latest

## Usage

chaotic-points list [–json] [packages…] chaotic-points lint [–rules f.json]… [–terms ‘s’]… [–terms-file f]… [–strict] [packages…]

`list` prints discovered points (`name  file:line`; `<dynamic>` for non-constant
names). `lint` exits non-zero when a rule references an explicit-point name that
does not exist; a glob matching no point is a warning (an error with `--strict`).

## CI gate

chaotic-points lint –rules chaos-rules.json ./…

Add it to CI to catch a typo'd point name before it silently disables a chaos test.
