module achat

go 1.19

require (
	github.com/cc14514/go-alibp2p v0.0.3-rc5
	github.com/cc14514/go-alibp2p-chat v0.0.0-20200321034458-351a53523aa8
	github.com/google/uuid v1.1.1
	github.com/multiformats/go-multihash v0.0.13 // indirect
	github.com/peterh/liner v1.1.0
	github.com/urfave/cli v1.22.2
	golang.org/x/net v0.15.0
)

replace github.com/libp2p/go-libp2p-kad-dht => github.com/cc14514/go-libp2p-kad-dht v0.0.3-rc1
replace github.com/libp2p/go-libp2p => github.com/cc14514/go-libp2p v0.0.3-rc4
replace github.com/libp2p/go-libp2p-circuit => github.com/cc14514/go-libp2p-circuit v0.0.3-rc0

replace github.com/cc14514/go-alibp2p-chat => ../../../go-alibp2p-chat
