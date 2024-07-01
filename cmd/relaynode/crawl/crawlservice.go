package crawl

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/cmd/relaynode/config"
	"github.com/Qitmeer/qng/meerevm/eth"
	"github.com/Qitmeer/qng/node/service"
	pcommon "github.com/Qitmeer/qng/p2p/common"
	qparams "github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/urfave/cli/v2"
	"net"
	"path"
	"strings"
	"time"
)

type CrawlService struct {
	service.Service
	cfg       *config.Config
	ctx       *cli.Context
	nodeKey   *ecdsa.PrivateKey
	localNode *enode.Node
	ecfg      *eth.Config
}

func (s *CrawlService) Start() error {
	if err := s.Service.Start(); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Start %s Crawl Service ...", s.ecfg.Node.Name))

	eth.InitLog(s.cfg.DebugLevel, s.cfg.DebugPrintOrigins)

	pk, err := pcommon.PrivateKey(s.cfg.DataDir, s.cfg.PrivateKey, 0600)
	if err != nil {
		return err
	}
	nk, err := pcommon.ToECDSAPrivKey(pk)
	if err != nil {
		return err
	}
	s.nodeKey = nk
	return s.discv4Crawl()
}

func (s *CrawlService) discv4Crawl() error {
	ctx := s.ctx
	nodesFile := getNodesFilePath(s.cfg.DataDir, s.ecfg.Node.Name)
	var inputSet nodeSet
	if common.FileExist(nodesFile) {
		is, err := loadNodesJSON(nodesFile)
		if err != nil {
			return err
		}
		inputSet = is
	}

	disc, err := s.startV4()
	if err != nil {
		return err
	}
	defer disc.Close()
	c := newCrawler(inputSet, disc, disc.RandomNodes())
	c.revalidateInterval = 10 * time.Minute
	output := c.run(ctx.Duration(crawlTimeoutFlag.Name))
	result, err := serviceFilter(output, s.ecfg)
	return writeNodesJSON(nodesFile, result)
}

// startV4 starts an ephemeral discovery V4 node.
func (s *CrawlService) startV4() (*discover.UDPv4, error) {
	ctx := s.ctx
	ln, config, err := s.makeDiscoveryConfig()
	if err != nil {
		return nil, err
	}
	s.localNode = ln.Node()
	socket, err := listen(ln, ctx.String(listenAddrFlag.Name))
	if err != nil {
		return nil, err
	}
	disc, err := discover.ListenV4(socket, ln, config)
	if err != nil {
		return nil, err
	}
	return disc, nil
}

func (s *CrawlService) makeDiscoveryConfig() (*enode.LocalNode, discover.Config, error) {
	ctx := s.ctx
	var cfg discover.Config
	cfg.PrivateKey = s.nodeKey

	if commandHasFlag(ctx, bootnodesFlag) {
		bn, err := parseBootnodes(ctx)
		if err != nil {
			return nil, cfg, err
		}
		cfg.Bootnodes = bn
	}

	dbpath := ctx.String(nodedbFlag.Name)
	db, err := enode.OpenDB(dbpath)
	if err != nil {
		return nil, cfg, err
	}
	ln := enode.NewLocalNode(db, cfg.PrivateKey)
	return ln, cfg, nil
}

func (s *CrawlService) Stop() error {
	if err := s.Service.Stop(); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Stop %s DNS Service", s.ecfg.Node.Name))
	return nil
}

func (s *CrawlService) Node() *enode.Node {
	return s.localNode
}

func NewCrawlService(cfg *config.Config, ecfg *eth.Config, ctx *cli.Context) *CrawlService {
	return &CrawlService{
		cfg:  cfg,
		ecfg: ecfg,
		ctx:  ctx,
	}
}

func listen(ln *enode.LocalNode, addr string) (*net.UDPConn, error) {
	if addr == "" {
		addr = "0.0.0.0:0"
	}
	socket, err := net.ListenPacket("udp4", addr)
	if err != nil {
		return nil, err
	}
	usocket := socket.(*net.UDPConn)
	uaddr := socket.LocalAddr().(*net.UDPAddr)
	if uaddr.IP.IsUnspecified() {
		ln.SetFallbackIP(net.IP{127, 0, 0, 1})
	} else {
		ln.SetFallbackIP(uaddr.IP)
	}
	ln.SetFallbackUDP(uaddr.Port)
	return usocket, nil
}

func parseBootnodes(ctx *cli.Context) ([]*enode.Node, error) {
	s := params.SepoliaBootnodes
	if ctx.IsSet(bootnodesFlag.Name) {
		input := ctx.String(bootnodesFlag.Name)
		if input == "" {
			return nil, nil
		}
		s = strings.Split(input, ",")
	}
	nodes := make([]*enode.Node, len(s))
	var err error
	for i, record := range s {
		nodes[i], err = parseNode(record)
		if err != nil {
			return nil, fmt.Errorf("invalid bootstrap node: %v", err)
		}
		log.Info("parse bootnode", nodes[i].IPAddr().String(), nodes[i].String())
	}
	log.Info("Finished bootnodes", "count", len(s))
	return nodes, nil
}

// parseNode parses a node record and verifies its signature.
func parseNode(source string) (*enode.Node, error) {
	if strings.HasPrefix(source, "enode://") {
		return enode.ParseV4(source)
	}
	r, err := parseRecord(source)
	if err != nil {
		return nil, err
	}
	return enode.New(enode.ValidSchemes, r)
}

// parseRecord parses a node record from hex, base64, or raw binary input.
func parseRecord(source string) (*enr.Record, error) {
	bin := []byte(source)
	if d, ok := decodeRecordHex(bytes.TrimSpace(bin)); ok {
		bin = d
	} else if d, ok := decodeRecordBase64(bytes.TrimSpace(bin)); ok {
		bin = d
	}
	var r enr.Record
	err := rlp.DecodeBytes(bin, &r)
	return &r, err
}

func decodeRecordHex(b []byte) ([]byte, bool) {
	if bytes.HasPrefix(b, []byte("0x")) {
		b = b[2:]
	}
	dec := make([]byte, hex.DecodedLen(len(b)))
	_, err := hex.Decode(dec, b)
	return dec, err == nil
}

func decodeRecordBase64(b []byte) ([]byte, bool) {
	if bytes.HasPrefix(b, []byte("enr:")) {
		b = b[4:]
	}
	dec := make([]byte, base64.RawURLEncoding.DecodedLen(len(b)))
	n, err := base64.RawURLEncoding.Decode(dec, b)
	return dec[:n], err == nil
}

func getNodesFilePath(dataDir string, node string) string {
	nfp := path.Join(dataDir, qparams.ActiveNetParams.Name, node)
	return path.Join(nfp, "nodes.json")
}

func serviceFilter(ns nodeSet, cfg *eth.Config) (nodeSet, error) {
	filter := forkid.NewStaticFilter(cfg.Eth.Genesis.Config, cfg.Eth.Genesis.ToBlock())

	f := func(n nodeJSON) bool {
		var eth struct {
			ForkID forkid.ID
			Tail   []rlp.RawValue `rlp:"tail"`
		}
		if n.N.Load(enr.WithEntry("eth", &eth)) != nil {
			return false
		}
		return filter(eth.ForkID) == nil
	}

	result := nodeSet{}
	for id, node := range ns {
		if f(node) {
			result[id] = node
		}
	}
	if len(ns) != len(result) {
		log.Debug("Filter nodes", "name", cfg.Node.Name, "src", len(ns), "result", len(result))
	}

	return result, nil
}
