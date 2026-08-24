module github.com/RomanAgaltsev/chaotic/adapter/aws

go 1.26

toolchain go1.26.6

require (
	github.com/RomanAgaltsev/chaotic v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.43.7
	github.com/aws/smithy-go v1.27.8
)

replace github.com/RomanAgaltsev/chaotic => ../..
