module dragonsnshit

go 1.21

require (
	github.com/consensys/gnark v0.0.0
	github.com/dgraph-io/badger/v3 v3.0.0
)

replace github.com/consensys/gnark => ./stubs/gnark

replace github.com/dgraph-io/badger/v3 => ./stubs/badger
