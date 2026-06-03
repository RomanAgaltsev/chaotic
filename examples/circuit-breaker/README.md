# circuit-breaker

A circuit breaker opens after chaos makes a dependency fail repeatedly.

`newEngine` installs an always-fire error rule on `OpHTTPClient`. The breaker
(threshold 3) opens after three failed calls; the remaining seven of ten
requests short-circuit without touching the dependency, so the rule fires
exactly three times.