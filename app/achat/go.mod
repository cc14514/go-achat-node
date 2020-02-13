module achat

go 1.13

require (
	github.com/cc14514/go-alibp2p v0.0.0-20191202085231-bb621d242f43
	github.com/cc14514/go-alibp2p-chat v0.0.0-20200211014832-dba6e1132c6a
	github.com/google/uuid v1.1.1
	github.com/peterh/liner v1.1.0
	github.com/urfave/cli v1.22.2
	golang.org/x/net v0.0.0-20190404232315-eb5bcb51f2a3
)

replace github.com/libp2p/go-libp2p-kad-dht => github.com/cc14514/go-libp2p-kad-dht v0.0.0-20191107040323-2463a62af156

replace github.com/libp2p/go-libp2p => github.com/cc14514/go-libp2p v0.0.0-20200118065341-58abd62e1061

replace github.com/libp2p/go-libp2p-swarm => github.com/cc14514/go-libp2p-swarm v0.0.0-20200118064831-601363b81fc2

replace github.com/libp2p/go-libp2p-circuit => github.com/cc14514/go-libp2p-circuit v0.0.0-20191111122236-413fc41ad3d7

replace github.com/cc14514/go-alibp2p => ../../../go-alibp2p

replace github.com/cc14514/go-alibp2p-chat => ../../../go-alibp2p-chat
