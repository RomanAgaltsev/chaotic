module github.com/ag4r/chaotic/examples/redis-cache-fallback

go 1.26

toolchain go1.26.4

require (
	github.com/ag4r/chaotic v0.0.0
	github.com/ag4r/chaotic/adapter/redis v0.0.0
	github.com/alicebob/miniredis/v2 v2.38.0
	github.com/redis/go-redis/v9 v9.20.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/ag4r/chaotic => ../..

replace github.com/ag4r/chaotic/adapter/redis => ../../adapter/redis
