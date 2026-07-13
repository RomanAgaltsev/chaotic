module github.com/RomanAgaltsev/chaotic/examples/grpc-stream-reconnect

go 1.26

require (
	github.com/RomanAgaltsev/chaotic v0.0.0
	github.com/RomanAgaltsev/chaotic/adapter/grpc v0.0.0
	google.golang.org/grpc v1.82.0
)

require (
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260523011958-0a33c5d7ca68 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/RomanAgaltsev/chaotic => ../..

replace github.com/RomanAgaltsev/chaotic/adapter/grpc => ../../adapter/grpc
