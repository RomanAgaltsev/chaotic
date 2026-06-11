// Package scenarios provides one-call chaos recipes for common failure modes.
// Each function attaches a coherent set of named rules (scenarios/<id>) to an
// engine using engine.NewRule, so the rules behave and report exactly like
// hand-written ones. Tune a scenario with up to three options; for anything more
// specific, drop down to engine.NewRule directly.
//
// Catalog:
//
//	DatabaseOutageCascade      - SQL/pgx connections drop, then recover slowly.
//	ThunderingHerdAfterDeploy  - a fraction of HTTP-server requests return 503.
//	SlowLeaderElection         - Redis (lock/coordination) calls run slow for a window.
//	PartialNetworkPartition    - a fraction of outbound gRPC/HTTP calls drop.
//
// AWSRegionFailover is pending: it needs engine.OpAWS (the adapter/aws kind),
// which lands with that adapter; it will be added here as an additive follow-on.
package scenarios
