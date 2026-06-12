module github.com/ag4r/chaotic/examples/slow-body-read

go 1.26

toolchain go1.26.4

require (
	github.com/ag4r/chaotic v0.0.0
	github.com/ag4r/chaotic/adapter/io v0.0.0
)

replace github.com/ag4r/chaotic => ../..

replace github.com/ag4r/chaotic/adapter/io => ../../adapter/io
