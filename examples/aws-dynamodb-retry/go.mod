module github.com/ag4r/chaotic/examples/aws-dynamodb-retry

go 1.26

toolchain go1.26.4

require (
	github.com/ag4r/chaotic v0.0.0
	github.com/ag4r/chaotic/adapter/aws v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.42.0
	github.com/aws/aws-sdk-go-v2/credentials v1.19.24
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.59.0
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.12.6 // indirect
	github.com/aws/smithy-go v1.27.2 // indirect
)

replace github.com/ag4r/chaotic => ../..

replace github.com/ag4r/chaotic/adapter/aws => ../../adapter/aws
