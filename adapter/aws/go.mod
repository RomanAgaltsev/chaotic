module github.com/ag4r/chaotic/adapter/aws

go 1.26

toolchain go1.26.4

require (
	github.com/ag4r/chaotic v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.42.0
	github.com/aws/smithy-go v1.27.2
)

replace github.com/ag4r/chaotic => ../..
