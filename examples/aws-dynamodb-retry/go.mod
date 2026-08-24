module github.com/RomanAgaltsev/chaotic/examples/aws-dynamodb-retry

go 1.26

toolchain go1.26.6

require (
	github.com/RomanAgaltsev/chaotic v0.0.0
	github.com/RomanAgaltsev/chaotic/adapter/aws v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.43.7
	github.com/aws/aws-sdk-go-v2/credentials v1.19.37
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.63.4
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.38 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.38 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.12.15 // indirect
	github.com/aws/smithy-go v1.27.8 // indirect
)

replace github.com/RomanAgaltsev/chaotic => ../..

replace github.com/RomanAgaltsev/chaotic/adapter/aws => ../../adapter/aws
