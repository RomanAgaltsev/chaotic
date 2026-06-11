module github.com/ag4r/chaotic/adapter/redis

go 1.26

toolchain go1.26.4

require (
	github.com/ag4r/chaotic v0.0.0-00010101000000-000000000000
	github.com/alicebob/miniredis/v2 v2.38.0
	github.com/redis/go-redis/v9 v9.20.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
)

replace github.com/ag4r/chaotic => ../..
