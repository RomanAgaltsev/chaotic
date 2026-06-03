# retry-http

A retry loop recovers from a transient failure injected into an
`http.Client` transport.

`newEngine` installs a `Times(1)` error rule on `OpHTTPClient`, so the **first**
request fails and the retry succeeds. `TestRetryRecovers` proves the retry path
works; `TestSingleAttemptSurfacesFault` proves the fault really fires.