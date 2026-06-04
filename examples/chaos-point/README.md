# chaos-point example

Demonstrates `chaos.PointWith` — an explicit injection point at a boundary no
adapter wraps (a post-commit publish hook), carrying an attribute bag (`id`)
that rules can match on via `engine.MatchAttr`.

`newEngine` installs a rule that fails the **first** `publish.afterCommit`
point with a transient error (`Times(1)`). `publishWithRetry` retries, so the
second attempt succeeds — proving the retry path actually recovers.

Run it:

    go run .

Test it:

    go test ./...

`TestRetrySurvivesInjectedFault` asserts the retry recovers;
`TestSingleAttemptFails` asserts the fault really fires (attempts=1 surfaces it).