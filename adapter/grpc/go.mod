module github.com/RomanAgaltsev/chaotic/adapter/grpc

go 1.26

toolchain go1.26.4

require (
	github.com/RomanAgaltsev/chaotic v0.0.0
	google.golang.org/grpc v1.81.1
)

require (
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.44.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260523011958-0a33c5d7ca68 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/RomanAgaltsev/chaotic => ../..
