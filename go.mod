module github.com/Qitmeer/qng

go 1.14

require (
	github.com/Qitmeer/crypto v0.0.0-20201028030128-6ed4040ca34a
	github.com/Qitmeer/crypto/cryptonight v0.0.0-20201028030128-6ed4040ca34a
	github.com/aristanetworks/goarista v0.0.0-20220314170124-2797d9e951fe
	github.com/cloudflare/roughtime v0.0.0-20210217223727-1fe56bcbcfd4
	github.com/davecgh/go-spew v1.1.1
	github.com/davidlazar/go-crypto v0.0.0-20190912175916-7055855a373f // indirect
	github.com/dchest/blake256 v1.1.0
	github.com/deckarep/golang-set v1.7.1
	github.com/dgraph-io/ristretto v0.0.2
	github.com/ethereum/go-ethereum v1.10.9
	github.com/ferranbt/fastssz v0.0.0-20200514094935-99fccaf93472
	github.com/go-stack/stack v1.8.1
	github.com/gogo/protobuf v1.3.2
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.4
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d
	github.com/ipfs/go-ds-leveldb v0.4.2
	github.com/ipfs/go-ipfs-addr v0.0.1
	github.com/jessevdk/go-flags v1.4.0
	github.com/jrick/logrotate v1.0.0
	github.com/libp2p/go-libp2p v0.11.0
	github.com/libp2p/go-libp2p-circuit v0.3.1
	github.com/libp2p/go-libp2p-core v0.6.1
	github.com/libp2p/go-libp2p-discovery v0.5.0
	github.com/libp2p/go-libp2p-kad-dht v0.5.0
	github.com/libp2p/go-libp2p-noise v0.1.1
	github.com/libp2p/go-libp2p-peerstore v0.2.6
	github.com/libp2p/go-libp2p-pubsub v0.3.2
	github.com/libp2p/go-libp2p-secio v0.2.2
	github.com/libp2p/go-sockaddr v0.1.0 // indirect
	github.com/magiconair/properties v1.8.5
	github.com/mattn/go-colorable v0.1.12
	github.com/minio/highwayhash v1.0.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multiaddr-net v0.2.0
	github.com/multiformats/go-multistream v0.1.2
	github.com/naoina/toml v0.1.2-0.20170918210437-9fafd6967416
	github.com/pkg/errors v0.9.1
	github.com/prysmaticlabs/go-bitfield v0.0.0-20200618145306-2ae0807bef65
	github.com/prysmaticlabs/go-ssz v0.0.0-20200101200214-e24db4d9e963
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/schollz/progressbar/v3 v3.8.3
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/urfave/cli/v2 v2.3.0
	github.com/zeromq/goczmq v4.1.0+incompatible
	go.opencensus.io v0.22.4
	golang.org/x/crypto v0.0.0-20211202192323-5770296d904e
	golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2
	golang.org/x/sys v0.0.0-20211204120058-94396e421777
	golang.org/x/tools v0.1.6
	gonum.org/v1/gonum v0.6.0
	gopkg.in/urfave/cli.v1 v1.20.0
	// indirect
	gopkg.in/yaml.v2 v2.4.0
)

replace (
	golang.org/x/crypto v0.0.0-20181001203147-e3636079e1a4 => github.com/golang/crypto v0.0.0-20181001203147-e3636079e1a4
	golang.org/x/net v0.0.0-20180906233101-161cd47e91fd => github.com/golang/net v0.0.0-20180906233101-161cd47e91fd
	golang.org/x/net v0.0.0-20181005035420-146acd28ed58 => github.com/golang/net v0.0.0-20181005035420-146acd28ed58
	golang.org/x/tools v0.0.0-20181006002542-f60d9635b16a => github.com/golang/tools v0.0.0-20181006002542-f60d9635b16a
)

replace github.com/ethereum/go-ethereum v1.10.9 => github.com/Qitmeer/go-ethereum v1.10.9-q.6
