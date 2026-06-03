module github.com/ag4r/chaotic/source/http

go 1.26

toolchain go1.26.4

require github.com/ag4r/chaotic v0.0.0

require (
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/ag4r/chaotic => ../..
