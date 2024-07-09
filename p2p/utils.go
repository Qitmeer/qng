package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"
	qprotocol "github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/iputils"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	"github.com/Qitmeer/qng/p2p/qnr"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/version"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/prysmaticlabs/go-bitfield"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"
	"time"
)

const metaDataPath = "metaData"
const PeerStore = "peerstore"

const dialTimeout = 1 * time.Second

// Retrieves an external ipv4 address and converts into a libp2p formatted value.
func IpAddr() net.IP {
	ip, err := iputils.ExternalIPv4()
	if err != nil {
		log.Error(fmt.Sprintf("Could not get IPv4 address: %v", err))
		return nil
	}
	return net.ParseIP(ip)
}

// Determines a private key for p2p networking from the p2p service's
// configuration struct. If no key is found, it generates a new one.
func privKey(cfg *common.Config) (crypto.PrivKey, error) {
	return common.PrivateKey(cfg.DataDir, cfg.PrivateKey, cfg.ReadWritePermissions)
}

func ConvertToInterfacePubkey(pubkey *ecdsa.PublicKey) crypto.PubKey {
	cpubKey, err := crypto.ECDSAPublicKeyFromPubKey(*pubkey)
	if err != nil {
		log.Error(err.Error())
	}
	return cpubKey
}

// SerializeQNR takes the qnr record in its key-value form and serializes it.
func SerializeQNR(record *qnr.Record) (string, error) {
	buf := bytes.NewBuffer([]byte{})
	if err := record.EncodeRLP(buf); err != nil {
		return "", fmt.Errorf("could not encode ENR record to bytes:%w", err)
	}
	enrString := base64.URLEncoding.EncodeToString(buf.Bytes())
	return enrString, nil
}

// Retrieves node p2p metadata from a set of configuration values
// from the p2p service.
func metaDataFromConfig(cfg *common.Config) (*pb.MetaData, error) {
	if len(cfg.DataDir) <= 0 && len(cfg.MetaDataDir) <= 0 {
		return &pb.MetaData{
			SeqNumber: 0,
			Subnets:   bitfield.NewBitvector64(),
		}, nil
	}
	defaultKeyPath := path.Join(cfg.DataDir, metaDataPath)
	metaDataPath := cfg.MetaDataDir

	_, err := os.Stat(defaultKeyPath)
	defaultMetadataExist := !os.IsNotExist(err)
	if err != nil && defaultMetadataExist {
		return nil, err
	}
	if metaDataPath == "" && !defaultMetadataExist {
		metaData := &pb.MetaData{
			SeqNumber: 0,
			Subnets:   bitfield.NewBitvector64(),
		}
		dst, err := metaData.Marshal()
		if err != nil {
			return nil, err
		}
		if err = ioutil.WriteFile(defaultKeyPath, dst, cfg.ReadWritePermissions); err != nil {
			return nil, err
		}
		return metaData, nil
	}
	if defaultMetadataExist && metaDataPath == "" {
		metaDataPath = defaultKeyPath
	}
	src, err := ioutil.ReadFile(metaDataPath)
	if err != nil {
		log.Error(fmt.Sprintf("Error reading metadata from file:%s", err.Error()))
		return nil, err
	}
	metaData := &pb.MetaData{}
	if err := metaData.Unmarshal(src); err != nil {
		return nil, err
	}
	return metaData, nil
}

// Attempt to dial an address to verify its connectivity
func verifyConnectivity(addr string, port uint, protocol string) (interface{}, error) {
	if addr != "" && len(protocol) > 0 {
		a := fmt.Sprintf("%s:%d", addr, port)
		conn, err := net.DialTimeout(protocol, a, dialTimeout)
		if err != nil {
			log.Warn(fmt.Sprintf("IP address is not accessible:protocol=%s address=%s error=%s", protocol, a, err))
			return nil, err
		}
		info := fmt.Sprintf("%s %s OK", conn.RemoteAddr().String(), conn.RemoteAddr().Network())
		err = conn.Close()
		if err != nil {
			log.Warn(fmt.Sprintf("Could not close connection:protocol=%s address=%s error=%s", protocol, a, err))
		}
		return info, nil
	}
	return nil, fmt.Errorf("input address is error")
}

func filterBootStrapAddrs(hostID string, addrs []string) []string {
	result := []string{}
	for _, addr := range addrs {
		if strings.HasSuffix(addr, hostID) {
			continue
		}
		result = append(result, addr)
	}
	return result
}

func BuildUserAgent(name string) string {
	return fmt.Sprintf("%s|%s|%s", name, version.String(), params.ActiveNetParams.Name)
}

func ProtocolDHT() protocol.ID {
	switch params.ActiveNetParams.Net {
	case qprotocol.MainNet:
		return "/qitmeer/kad/1.0.0"
	case qprotocol.TestNet:
		return "/qitmeer/kad/1.0.0/testnet"
	case qprotocol.MixNet:
		return "/qitmeer/kad/1.0.0/mixnet"
	default:
		return "/qitmeer/kad/1.0.0/privnet"
	}
}

func Cleanup(dataDir string) {
	dsPath := path.Join(dataDir, PeerStore)
	err := os.RemoveAll(dsPath)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info(fmt.Sprintf("Finished cleanup:%s", dsPath))
}
