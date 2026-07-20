module github.com/RomanAgaltsev/chaotic/examples/pgx-pool

go 1.26

require (
	github.com/RomanAgaltsev/chaotic v0.0.0
	github.com/RomanAgaltsev/chaotic/adapter/pgx v0.0.0
	github.com/jackc/pgx/v5 v5.10.0
)

require (
	github.com/RomanAgaltsev/chaotic/adapter/net v1.2.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.40.0 // indirect
)

replace github.com/RomanAgaltsev/chaotic => ../..

replace github.com/RomanAgaltsev/chaotic/adapter/pgx => ../../adapter/pgx
