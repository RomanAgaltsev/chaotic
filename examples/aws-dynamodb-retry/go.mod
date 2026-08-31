module github.com/RomanAgaltsev/chaotic/examples/aws-dynamodb-retry

go 1.26

toolchain go1.26.6

require (
	github.com/RomanAgaltsev/chaotic v0.0.0
	github.com/RomanAgaltsev/chaotic/adapter/aws v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.45.1
	github.com/aws/aws-sdk-go-v2/credentials v1.20.1
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.65.1
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.13.1 // indirect
	github.com/aws/smithy-go v1.28.1 // indirect
)

replace github.com/RomanAgaltsev/chaotic => ../..

replace github.com/RomanAgaltsev/chaotic/adapter/aws => ../../adapter/aws
