module dragonsnshit

go 1.21

require (
	github.com/consensys/gnark v0.0.0
	github.com/dgraph-io/badger/v3 v3.0.0
	golang.org/x/net v0.26.0
)

require golang.org/x/crypto v0.24.0 // indirect

replace github.com/consensys/gnark => ./stubs/gnark

replace github.com/dgraph-io/badger/v3 => ./stubs/badger
