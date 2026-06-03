# db-conn-pool

`database/sql` discards a poisoned connection and retries on a fresh one.

`newEngine` installs a `Times(1)` `ConnDrop` rule on `OpSQL`. The adapter maps
`ConnDrop` to `driver.ErrBadConn`, which makes `database/sql` discard that
connection and retry on a new one — so a single `Exec` succeeds while the
fake driver opens two connections.