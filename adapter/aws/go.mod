module github.com/RomanAgaltsev/chaotic/adapter/aws

go 1.26.0

toolchain go1.26.6

require (
	github.com/RomanAgaltsev/chaotic v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.45.1
	github.com/aws/smithy-go v1.28.1
)

replace github.com/RomanAgaltsev/chaotic => ../..
