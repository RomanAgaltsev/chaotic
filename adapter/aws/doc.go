// Package aws is a chaos adapter for the AWS SDK for Go v2. Every service client
// (DynamoDB, S3, SQS, Lambda, ...) runs the same smithy-go middleware stack, so a
// single chaos middleware covers them all. Append it to your aws.Config and any
// client built from that config consults the chaotic engine on each request:
//
//	import chaosaws "github.com/ag4r/chaotic/adapter/aws"
//
//	cfg, _ := config.LoadDefaultConfig(ctx)
//	chaosaws.AppendChaosMiddleware(&cfg, eng)
//	ddb := dynamodb.NewFromConfig(cfg) // chaos-aware
//
// The middleware runs at the Finalize step by default (after retry setup, before
// send), so an injected error is classified and retried by the SDK exactly like a
// real failure. AppendChaosMiddlewareWith lets you choose StepBuild to fault
// before signing/retry classification instead.
//
// The Op is built from request-context metadata only (service, operation, region)
// — never from the request payload.
//
//	engine.Op{Kind: OpAWS, Name: "dynamodb.GetItem", Method: "request",
//	          Attrs: {"service": "dynamodb", "operation": "GetItem", "region": "us-east-1"}}
//
// Fault mapping:
//
//	fault.Latency / Jittered  -> ctx-honoring sleep before send
//	fault.Error(err)          -> err is returned as-is (supply &smithy.GenericAPIError{...} for realism)
//	fault.ConnDrop()          -> *net.OpError, which the SDK's retryer treats as a retryable connection error
//	fault.Panic(v)            -> panic(v)
//
// Build with -tags chaos_off to compile the adapter out: AppendChaosMiddleware
// becomes a no-op and registers no middleware.
package aws
