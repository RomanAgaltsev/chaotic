# chaotic-points

Static analyzer for chaotic explicit injection points. Discovers every
`chaos.Point` / `chaos.PointWith` call site in a module and gates a rules config
against typo'd point names, so a rule that targets a nonexistent point fails CI.

## Install

```bash
go install github.com/RomanAgaltsev/chaotic/cmd/chaotic-points@latest
```

## Usage

```bash
chaotic-points list [--json] [-C dir] [packages...]
chaotic-points lint [--rules f.json]... [--terms s]... [--terms-file f]... [--strict] [-C dir] [packages...]
```

`list` prints discovered points (`name  file:line`; `<dynamic>` for non-constant
names). `lint` exits non-zero when a rule references an explicit-point name that
does not exist; a glob matching no point is a warning (an error with `--strict`).
Pass `-C dir` to scan a module rooted elsewhere than the current directory.

## CI gate

```bash
chaotic-points lint --rules chaos-rules.json ./...
```

Add it to CI to catch a typo'd point name before it silently disables a chaos test.
