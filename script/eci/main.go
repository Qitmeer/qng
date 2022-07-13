package main

import (
	"fmt"
	"github.com/Unknwon/goconfig"
	eci "github.com/alibabacloud-go/eci-20180808/client"
	rpcconfig "github.com/alibabacloud-go/tea-rpc/client"
	"os"
	"strconv"
	"strings"
)

var client *eci.Client

var accessKey string
var secretKey string
var regionId string
var zoneId string
var securityGroupId string
var nfsServer string
var vSwitchId string
var qngImage string
var dockerContainerCount int
var endpoint string
var cpuCores = float32(2.0)
var memCores = float32(4.0)
var dataDirPrefix string
var dockerDataDir string
var dockerExecCommand string
var containerName string
var dockerExecArgs []string
var exiprePeriod int64

func init() {
	var cfg *goconfig.ConfigFile
	config, err := goconfig.LoadConfigFile("./config.conf")
	if err != nil {
		fmt.Println("get config file error:", err.Error())
		os.Exit(-1)
	}
	cfg = config
	accessKey, _ = cfg.GetValue("eci_conf", "access_key")
	secretKey, _ = cfg.GetValue("eci_conf", "secret_key")
	regionId, _ = cfg.GetValue("eci_conf", "region_id")
	zoneId, _ = cfg.GetValue("eci_conf", "zone_id")
	securityGroupId, _ = cfg.GetValue("eci_conf", "security_group_id")
	vSwitchId, _ = cfg.GetValue("eci_conf", "v_switch_id")
	vSwitchId, _ = cfg.GetValue("eci_conf", "v_switch_id")
	qngImage, _ = cfg.GetValue("eci_conf", "qng_image")
	endpoint, _ = cfg.GetValue("eci_conf", "endpoint")
	cs, _ := cfg.GetValue("eci_conf", "docker_container_count")
	dockerContainerCount, _ = strconv.Atoi(cs)
	nfsServer, _ = cfg.GetValue("eci_conf", "nfs_server")
	dataDirPrefix, _ = cfg.GetValue("eci_conf", "data_dir_prefix")
	containerName, _ = cfg.GetValue("eci_conf", "container_name")
	dockerDataDir, _ = cfg.GetValue("eci_conf", "docker_data_dir")
	dockerExecCommand, _ = cfg.GetValue("eci_conf", "docker_exec_command")
	ep, _ := cfg.GetValue("eci_conf", "expire_period")
	exiprePeriod, _ = strconv.ParseInt(ep, 10, 64)
	args, _ := cfg.GetValue("eci_conf", "docker_exec_args")

	dockerExecArgs = strings.Split(args, ",")
	fmt.Printf("init success[ access_key:%s, secret_key:%s, region_id:%s, "+
		"zoneId:%s, vSwitchId:%s, securityGroupId:%s ,ContainerCount:%d,expirePeriod:%d]\n",
		accessKey, secretKey, regionId, zoneId, vSwitchId, securityGroupId, dockerContainerCount, exiprePeriod)

	//init eci client
	// init config
	var eci_config = new(rpcconfig.Config).SetAccessKeyId(accessKey).
		SetAccessKeySecret(secretKey).
		SetRegionId(regionId).
		SetEndpoint(endpoint).
		SetType("access_key")

	// init client
	client, err = eci.NewClient(eci_config)
	if err != nil {
		panic(err)
	}

}

func main() {
	for i := 0; i < dockerContainerCount; i++ {
		CreateContainerGroup(i)
	}
}
