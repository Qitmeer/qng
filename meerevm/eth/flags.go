package eth

import (
	mcommon "github.com/Qitmeer/qng/v2/meerevm/common"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/urfave/cli/v2"
)

func GetFlags() []cli.Flag {

	var (
		nodeFlags = mcommon.Merge([]cli.Flag{
			utils.IdentityFlag,
			utils.UnlockedAccountFlag,
			utils.PasswordFileFlag,
			utils.BootnodesFlag,
			utils.MinFreeDiskSpaceFlag,
			utils.KeyStoreDirFlag,
			utils.ExternalSignerFlag,
			utils.NoUSBFlag, // deprecated
			utils.USBFlag,
			utils.SmartCardDaemonPathFlag,
			utils.OverrideOsaka,
			utils.OverrideVerkle,
			utils.EnablePersonal, // deprecated
			utils.TxPoolLocalsFlag,
			utils.TxPoolNoLocalsFlag,
			utils.TxPoolJournalFlag,
			utils.TxPoolRejournalFlag,
			utils.TxPoolPriceLimitFlag,
			utils.TxPoolPriceBumpFlag,
			utils.TxPoolAccountSlotsFlag,
			utils.TxPoolGlobalSlotsFlag,
			utils.TxPoolAccountQueueFlag,
			utils.TxPoolGlobalQueueFlag,
			utils.TxPoolLifetimeFlag,
			utils.BlobPoolDataDirFlag,
			utils.BlobPoolDataCapFlag,
			utils.BlobPoolPriceBumpFlag,
			utils.SyncModeFlag,
			utils.SyncTargetFlag,
			utils.ExitWhenSyncedFlag,
			utils.GCModeFlag,
			utils.SnapshotFlag,
			utils.TxLookupLimitFlag, // deprecated
			utils.TransactionHistoryFlag,
			utils.ChainHistoryFlag,
			utils.LogHistoryFlag,
			utils.LogNoHistoryFlag,
			utils.LogExportCheckpointsFlag,
			utils.StateHistoryFlag,
			utils.LightKDFFlag,
			utils.EthRequiredBlocksFlag,
			utils.LegacyWhitelistFlag, // deprecated
			utils.CacheFlag,
			utils.CacheDatabaseFlag,
			utils.CacheTrieFlag,
			utils.CacheTrieJournalFlag,   // deprecated
			utils.CacheTrieRejournalFlag, // deprecated
			utils.CacheGCFlag,
			utils.CacheSnapshotFlag,
			utils.CacheNoPrefetchFlag,
			utils.CachePreimagesFlag,
			utils.CacheLogSizeFlag,
			utils.FDLimitFlag,
			utils.CryptoKZGFlag,
			utils.ListenPortFlag,
			utils.DiscoveryPortFlag,
			utils.MaxPeersFlag,
			utils.MaxPendingPeersFlag,
			utils.MiningEnabledFlag, // deprecated
			utils.MinerGasLimitFlag,
			utils.MinerGasPriceFlag,
			utils.MinerEtherbaseFlag, // deprecated
			utils.MinerExtraDataFlag,
			utils.MinerRecommitIntervalFlag,
			utils.MinerPendingFeeRecipientFlag,
			utils.MinerNewPayloadTimeoutFlag, // deprecated
			utils.NATFlag,
			utils.NoDiscoverFlag,
			utils.DiscoveryV4Flag,
			utils.DiscoveryV5Flag,
			utils.LegacyDiscoveryV5Flag, // deprecated
			utils.NetrestrictFlag,
			utils.NodeKeyFileFlag,
			utils.NodeKeyHexFlag,
			utils.DNSDiscoveryFlag,
			utils.DeveloperFlag,
			utils.DeveloperGasLimitFlag,
			utils.DeveloperPeriodFlag,
			utils.VMEnableDebugFlag,
			utils.VMTraceFlag,
			utils.VMTraceJsonConfigFlag,
			utils.NetworkIdFlag,
			utils.EthStatsURLFlag,
			utils.GpoBlocksFlag,
			utils.GpoPercentileFlag,
			utils.GpoMaxGasPriceFlag,
			utils.GpoIgnoreGasPriceFlag,
			ConfigFileFlag,
			utils.LogDebugFlag,
			utils.LogBacktraceAtFlag,
			utils.BeaconApiFlag,
			utils.BeaconApiHeaderFlag,
			utils.BeaconThresholdFlag,
			utils.BeaconNoFilterFlag,
			utils.BeaconConfigFlag,
			utils.BeaconGenesisRootFlag,
			utils.BeaconGenesisTimeFlag,
			utils.BeaconCheckpointFlag,
			utils.BeaconCheckpointFileFlag,
		}, utils.NetworkFlags, utils.DatabaseFlags)

		rpcFlags = []cli.Flag{
			utils.HTTPEnabledFlag,
			utils.HTTPListenAddrFlag,
			utils.HTTPPortFlag,
			utils.HTTPCORSDomainFlag,
			utils.AuthListenFlag,
			utils.AuthPortFlag,
			utils.AuthVirtualHostsFlag,
			utils.JWTSecretFlag,
			utils.HTTPVirtualHostsFlag,
			utils.GraphQLEnabledFlag,
			utils.GraphQLCORSDomainFlag,
			utils.GraphQLVirtualHostsFlag,
			utils.HTTPApiFlag,
			utils.HTTPPathPrefixFlag,
			utils.WSEnabledFlag,
			utils.WSListenAddrFlag,
			utils.WSPortFlag,
			utils.WSApiFlag,
			utils.WSAllowedOriginsFlag,
			utils.WSPathPrefixFlag,
			utils.IPCDisabledFlag,
			utils.IPCPathFlag,
			utils.InsecureUnlockAllowedFlag,
			utils.RPCGlobalGasCapFlag,
			utils.RPCGlobalEVMTimeoutFlag,
			utils.RPCGlobalTxFeeCapFlag,
			utils.AllowUnprotectedTxs,
			utils.BatchRequestLimit,
			utils.BatchResponseMaxSize,
		}

		metricsFlags = []cli.Flag{
			utils.MetricsEnabledFlag,
			utils.MetricsEnabledExpensiveFlag,
			utils.MetricsHTTPFlag,
			utils.MetricsPortFlag,
			utils.MetricsEnableInfluxDBFlag,
			utils.MetricsInfluxDBEndpointFlag,
			utils.MetricsInfluxDBDatabaseFlag,
			utils.MetricsInfluxDBUsernameFlag,
			utils.MetricsInfluxDBPasswordFlag,
			utils.MetricsInfluxDBTagsFlag,
			utils.MetricsEnableInfluxDBV2Flag,
			utils.MetricsInfluxDBTokenFlag,
			utils.MetricsInfluxDBBucketFlag,
			utils.MetricsInfluxDBOrganizationFlag,
		}
	)

	flags := []cli.Flag{}
	flags = append(flags, nodeFlags...)
	flags = append(flags, rpcFlags...)
	flags = append(flags, metricsFlags...)
	return flags
}
