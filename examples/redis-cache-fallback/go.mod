module github.com/RomanAgaltsev/chaotic/examples/redis-cache-fallback

go 1.26

toolchain go1.26.5

require (
	github.com/RomanAgaltsev/chaotic v0.0.0
	github.com/RomanAgaltsev/chaotic/adapter/redis v0.0.0
	github.com/alicebob/miniredis/v2 v2.38.0
	github.com/redis/go-redis/v9 v9.21.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/RomanAgaltsev/chaotic => ../..

replace github.com/RomanAgaltsev/chaotic/adapter/redis => ../../adapter/redis
