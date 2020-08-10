package main

import (
	"context"
	"fmt"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	config "github.com/ipfs/go-ipfs-config"
	files "github.com/ipfs/go-ipfs-files"
	libp2p "github.com/ipfs/go-ipfs/core/node/libp2p"
	icore "github.com/ipfs/interface-go-ipfs-core"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/plugin/loader" // This package is needed so that all the preloaded plugins are loaded automatically
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/libp2p/go-libp2p-core/peer"
)

/// ------ Setting up the IPFS Repo

func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}



	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

func createTempRepo(ctx context.Context, reponame string) (string, error) {
	repoPath, err := ioutil.TempDir("", "ipfs-shell-" + reponame)
	if err != nil {
		return "", fmt.Errorf("failed to get temp dir: %s", err)
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(ioutil.Discard, 2048)
	if err != nil {
		return "", err
	}

	// Create the repo with the config
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to init ephemeral node: %s", err)
	}

	return repoPath, nil
}

/// ------ Spawning the node

// Creates an IPFS node and returns its coreAPI
func createNode(ctx context.Context, repoPath string) (icore.CoreAPI, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	// Construct the node

	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, err
	}

	// Attach the Core API to the constructed node
	return coreapi.NewCoreAPI(node)
}

// Spawns a node on the default repo location, if the repo exists
func spawnDefault(ctx context.Context, ipfsPath string) (icore.CoreAPI, error) {
	//defaultPath, err := config.PathRoot()
	//if err != nil {
		// shouldn't be possible
		//return nil, err
	//}

	if err := setupPlugins(ipfsPath); err != nil {
		//return nil,err
		//if (err.Error() != "error initializing plugins: already have a datastore named \"badgerds\"") {
		//	return nil, err
		//}

	}

	return createNode(ctx, ipfsPath)
}

// Spawns a node to be used just for this run (i.e. creates a tmp repo)
func spawnEphemeral(ctx context.Context, reponame string) (icore.CoreAPI, error) {
	if err := setupPlugins("./" + reponame); err != nil {
		if (err.Error() != "error initializing plugins: already have a datastore named \"badgerds\"") {
			return nil, err
		}
	}

	// Create a Temporary Repo
	repoPath, err := createTempRepo(ctx,reponame)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp repo: %s", err)
	}

	// Spawning an ephemeral IPFS node
	return createNode(ctx, repoPath)
}

//

func connectToPeers(ctx context.Context, ipfs icore.CoreAPI, peers []string) error {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peerstore.PeerInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		pii, err := peerstore.InfoFromP2pAddr(addr)
		if err != nil {
			return err
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peerstore.PeerInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}

	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peerstore.PeerInfo) {
			defer wg.Done()
			err := ipfs.Swarm().Connect(ctx, *peerInfo)
			if err != nil {
				log.Printf("failed to connect to %s: %s", peerInfo.ID, err)
			}
		}(peerInfo)
	}
	wg.Wait()
	return nil
}

func getUnixfsFile(path string) (files.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	st, err := file.Stat()
	if err != nil {
		return nil, err
	}

	f, err := files.NewReaderPathFile(path, file, st)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func getUnixfsNode(path string) (files.Node, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := files.NewSerialFile(path, false, st)
	if err != nil {
		return nil, err
	}

	return f, nil
}

/// -------

func main() {
	/// --- Part I: Getting a IPFS node running

	fmt.Println("-- Getting an IPFS node running -- ")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()


	// Spawn a node using the default path (~/.ipfs), assuming that a repo exists there already

	ipfs1, err := spawnDefault(ctx,"~/.ipfs1")
	if err != nil {
		fmt.Println("Error opening repo ipfs1")
	}

	ipfs2, err := spawnDefault(ctx,"~/.ipfs2")
	if err != nil {
		fmt.Println("Error opening repo ipfs1")
	}

	ipfs3, err := spawnDefault(ctx,"~/.ipfs3")
	if err != nil {
		fmt.Println("Error opening repo ipfs1")
	}

	if ipfs1 == nil || ipfs2 == nil || ipfs3 == nil {
		panic(fmt.Errorf("failed to spawn one of the nodes"))
	}

	// Spawn a node using a temporary path, creating a temporary repo for the run
	fmt.Println("Spawning node on a temporary repo")

	//ipfs1, err := spawnEphemeral(ctx,"repo1")
	//if err != nil {
	//	panic(fmt.Errorf("failed to spawn ephemeral node1: %s", err))
	//}

	//ipfs2, err := spawnEphemeral(ctx, "repo2")
	//if err != nil {
	//	panic(fmt.Errorf("failed to spawn ephemeral node2: %s", err))
	//}
	//
	//ipfs3, err := spawnEphemeral(ctx,"repo3")
	//if err != nil {
	//	panic(fmt.Errorf("failed to spawn ephemeral node3: %s", err))
	//}

	fmt.Println("IPFS nodes are running")

	/// --- Part II: Adding a file and a directory to IPFS

	fmt.Println("\n-- Adding and getting back files & directories --")

	inputBasePath := "./example-folder/"
	inputPathFile := inputBasePath + "TorCoin.pdf"
	inputPathDirectory := inputBasePath + "test-dir"

	someFile, err := getUnixfsNode(inputPathFile)
	if err != nil {
		panic(fmt.Errorf("Could not get File: %s", err))
	}

	cidFile, err := ipfs1.Unixfs().Add(ctx, someFile)
	if err != nil {
		panic(fmt.Errorf("Could not add File: %s", err))
	}

	fmt.Printf("Added file to IPFS with CID %s\n", cidFile.String())

	someDirectory, err := getUnixfsNode(inputPathDirectory)
	if err != nil {
		panic(fmt.Errorf("Could not get File: %s", err))
	}

	cidDirectory, err := ipfs1.Unixfs().Add(ctx, someDirectory)
	if err != nil {
		panic(fmt.Errorf("Could not add Directory: %s", err))
	}

	fmt.Printf("Added directory to IPFS with CID %s\n", cidDirectory.String())

	// Verify connectivity
	ipfs1_local,err := ipfs1.Swarm().LocalAddrs(context.Background());
	if err != nil {
		panic(fmt.Errorf("Could not get local address for ipfs1: %s", err))
	}
	_ = ipfs1_local

	ipfs2_local,err := ipfs2.Swarm().LocalAddrs(context.Background());
	if err != nil {
		panic(fmt.Errorf("Could not get local address for ipfs2: %s", err))
	}

	for _,x := range ipfs2_local {
		print("Address: " + x.String())
	}
	_ = ipfs2_local

	ipfs1_peers, err := ipfs1.Swarm().Peers(context.Background())
	if err != nil {
		panic(fmt.Errorf("Could not get peers for ipfs1: %s", err))
	}
	_ = ipfs1_peers

	pinned, err := ipfs2.Pin().Ls(context.Background())

	for p := range pinned {
		ipfs2.Pin().Rm(context.Background(),p.Path())
		p.Path()
	}


	filePath := "" + cidFile.String()

	block, err := ipfs2.Block().Get(context.Background(), path.New(filePath))

	if err == nil && block != nil {
		ipfs2.Block().Rm(context.Background(), path.New(filePath),options.Block.Force(true))
	}

	//ipfs2_peers, err := ipfs2.Swarm().Peers(context.Background())
	//if err != nil {
	//	panic(fmt.Errorf("Could not get peers for ipfs2: %s", err))
	//}
	//
	//
	//ipfs3_peers, err := ipfs3.Swarm().Peers(context.Background())
	//if err != nil {
	//	panic(fmt.Errorf("Could not get peers for ipfs3: %s", err))
	//}

	/// --- Part III: Getting the file and directory you added back

	outputBasePath := "./example-folder/"
	outputPathFile := outputBasePath + strings.Split(cidFile.String(), "/")[2]
	outputPathDirectory := outputBasePath + strings.Split(cidDirectory.String(), "/")[2]

	rootNodeFile, err := ipfs2.Unixfs().Get(ctx, cidFile)
	if err != nil {
		panic(fmt.Errorf("Could not get file with CID: %s", err))
	}

	err = files.WriteTo(rootNodeFile, outputPathFile)
	if err != nil {
		panic(fmt.Errorf("Could not write out the fetched CID: %s", err))
	}

	fmt.Printf("Got file back from IPFS (IPFS path: %s) and wrote it to %s\n", cidFile.String(), outputPathFile)

	rootNodeDirectory, err := ipfs3.Unixfs().Get(ctx, cidDirectory)
	if err != nil {
		panic(fmt.Errorf("Could not get file with CID: %s", err))
	}


	err = files.WriteTo(rootNodeDirectory, outputPathDirectory)
	if err != nil {
		panic(fmt.Errorf("Could not write out the fetched CID: %s", err))
	}

	fmt.Printf("Got directory back from IPFS (IPFS path: %s) and wrote it to %s\n", cidDirectory.String(), outputPathDirectory)

	/// --- Part IV: Getting a file from the IPFS Network

	//fmt.Println("\n-- Going to connect to a few nodes in the Network as bootstrappers --")
	//
	//bootstrapNodes := []string{
	//	// IPFS Bootstrapper nodes.
	//	"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	//	"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	//	"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	//	"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	//
	//	// IPFS Cluster Pinning nodes
	//	"/ip4/138.201.67.219/tcp/4001/p2p/QmUd6zHcbkbcs7SMxwLs48qZVX3vpcM8errYS7xEczwRMA",
	//	"/ip4/138.201.67.219/udp/4001/quic/p2p/QmUd6zHcbkbcs7SMxwLs48qZVX3vpcM8errYS7xEczwRMA",
	//	"/ip4/138.201.67.220/tcp/4001/p2p/QmNSYxZAiJHeLdkBg38roksAR9So7Y5eojks1yjEcUtZ7i",
	//	"/ip4/138.201.67.220/udp/4001/quic/p2p/QmNSYxZAiJHeLdkBg38roksAR9So7Y5eojks1yjEcUtZ7i",
	//	"/ip4/138.201.68.74/tcp/4001/p2p/QmdnXwLrC8p1ueiq2Qya8joNvk3TVVDAut7PrikmZwubtR",
	//	"/ip4/138.201.68.74/udp/4001/quic/p2p/QmdnXwLrC8p1ueiq2Qya8joNvk3TVVDAut7PrikmZwubtR",
	//	"/ip4/94.130.135.167/tcp/4001/p2p/QmUEMvxS2e7iDrereVYc5SWPauXPyNwxcy9BXZrC1QTcHE",
	//	"/ip4/94.130.135.167/udp/4001/quic/p2p/QmUEMvxS2e7iDrereVYc5SWPauXPyNwxcy9BXZrC1QTcHE",
	//
	//	// You can add more nodes here, for example, another IPFS node you might have running locally, mine was:
	//	// "/ip4/127.0.0.1/tcp/4010/p2p/QmZp2fhDLxjYue2RiUvLwT9MWdnbDxam32qYFnGmxZDh5L",
	//	// "/ip4/127.0.0.1/udp/4010/quic/p2p/QmZp2fhDLxjYue2RiUvLwT9MWdnbDxam32qYFnGmxZDh5L",
	//}
	//
	//go connectToPeers(ctx, ipfs1, bootstrapNodes)
	//
	//exampleCIDStr := "QmUaoioqU7bxezBQZkUcgcSyokatMY71sxsALxQmRRrHrj"
	//
	//fmt.Printf("Fetching a file from the network with CID %s\n", exampleCIDStr)
	//outputPath := outputBasePath + exampleCIDStr
	//testCID := icorepath.New(exampleCIDStr)
	//
	//rootNode, err := ipfs1.Unixfs().Get(ctx, testCID)
	//if err != nil {
	//	panic(fmt.Errorf("Could not get file with CID: %s", err))
	//}
	//
	//err = files.WriteTo(rootNode, outputPath)
	//if err != nil {
	//	panic(fmt.Errorf("Could not write out the fetched CID: %s", err))
	//}
	//
	//fmt.Printf("Wrote the file to %s\n", outputPath)
	//
	//fmt.Println("\nAll done! You just finalized your first tutorial on how to use go-ipfs as a library")
}