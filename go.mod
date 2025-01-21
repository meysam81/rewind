module github.com/meysam81/requestrewind

go 1.23.3

replace github.com/meysam81/requestrewind => ./

require (
	github.com/lib/pq v1.10.9
	github.com/rs/zerolog v1.33.0
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/oklog/ulid/v2 v2.1.0
	golang.org/x/sys v0.12.0 // indirect
)
