module github.com/Qitmeer/meerevm

go 1.16

require github.com/ethereum/go-ethereum v1.10.9

require (
	github.com/Qitmeer/crypto v0.0.0-20201028030128-6ed4040ca34a // indirect
	github.com/Qitmeer/crypto/cryptonight v0.0.0-20201028030128-6ed4040ca34a // indirect
	github.com/Qitmeer/qng-core v1.2.7
	github.com/cloudflare/roughtime v0.0.0-20210217223727-1fe56bcbcfd4 // indirect
	github.com/dchest/blake256 v1.1.0 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	golang.org/x/crypto v0.0.0-20211202192323-5770296d904e
	gopkg.in/urfave/cli.v1 v1.20.0
)

replace github.com/ethereum/go-ethereum v1.10.9 => github.com/Qitmeer/go-ethereum v1.10.9-3

