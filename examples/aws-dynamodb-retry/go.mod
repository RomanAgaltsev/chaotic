module github.com/RomanAgaltsev/chaotic/examples/aws-dynamodb-retry

go 1.26

toolchain go1.26.5

require (
	github.com/RomanAgaltsev/chaotic v0.0.0
	github.com/RomanAgaltsev/chaotic/adapter/aws v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.42.1
	github.com/aws/aws-sdk-go-v2/credentials v1.19.26
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.59.2
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.12.7 // indirect
	github.com/aws/smithy-go v1.27.3 // indirect
)

replace github.com/RomanAgaltsev/chaotic => ../..

replace github.com/RomanAgaltsev/chaotic/adapter/aws => ../../adapter/aws
