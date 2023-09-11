module github.com/cc14514/go-alibp2p-chat

go 1.13

require (
	github.com/cc14514/go-alibp2p v0.0.3-rc5
	github.com/google/uuid v1.1.1
	github.com/syndtr/goleveldb v1.0.0
	github.com/tendermint/go-amino v0.0.0-20200130113325-59d50ef176f6
	golang.org/x/net v0.15.0
)

replace github.com/libp2p/go-libp2p-kad-dht => github.com/cc14514/go-libp2p-kad-dht v0.0.3-rc1
replace github.com/libp2p/go-libp2p => github.com/cc14514/go-libp2p v0.0.3-rc4
replace github.com/libp2p/go-libp2p-circuit => github.com/cc14514/go-libp2p-circuit v0.0.3-rc0

