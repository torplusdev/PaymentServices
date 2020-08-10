module paidpiper.com/ipfs/protocol

go 1.14

replace github.com/libp2p/go-libp2p-kad-dht => ../../go-libp2p-kad-dht
replace paidpiper.com/libp2p/go-libp2p-core => ../../go-libp2p-core
replace github.com/libp2p/go-libp2p-core => ../../go-libp2p-core

require (
	github.com/ipfs/go-ipfs v0.6.0
	github.com/ipfs/go-ipfs-config v0.8.0
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/interface-go-ipfs-core v0.3.0
	github.com/libp2p/go-libp2p-core v0.5.7
	github.com/libp2p/go-libp2p-peerstore v0.2.6
	github.com/multiformats/go-multiaddr v0.2.2
)
