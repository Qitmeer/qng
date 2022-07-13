package main

import (
	"fmt"
	"github.com/Unknwon/goconfig"
	eci "github.com/alibabacloud-go/eci-20180808/client"
	rpcconfig "github.com/alibabacloud-go/tea-rpc/client"
	"github.com/alibabacloud-go/tea/tea"
	"log"
	"os"
)

var client2 *eci.Client

var accessKey string
var secretKey string
var regionId string
var zoneId string
var securityGroupId string
var vSwitchId string
var qngImage string

/**
获取配置信息
*/
func init() {
	var cfg *goconfig.ConfigFile
	config, err := goconfig.LoadConfigFile("./config.conf") //加载配置文件
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

	fmt.Printf("init success[ access_key:%s, secret_key:%s, region_id:%s, zoneId:%s, vSwitchId:%s, securityGroupId:%s]\n",
		accessKey, secretKey, regionId, zoneId, vSwitchId, securityGroupId)

	//init eci client
	// init config
	var eci_config = new(rpcconfig.Config).SetAccessKeyId(accessKey).
		SetAccessKeySecret(secretKey).
		SetRegionId("cn-hangzhou").
		SetEndpoint("eci.aliyuncs.com").
		SetType("access_key")

	// init client
	client2, err = eci.NewClient(eci_config)
	if err != nil {
		panic(err)
	}

}

func main() {
	createContainerGroup_v2()
}

func createContainerGroup_v2() {
	// init runtimeObject
	//runtimeObject := new(util.RuntimeOptions).SetAutoretry(false).
	//	SetMaxIdleConns(3)

	// init request
	request := new(eci.CreateContainerGroupRequest)
	request.SetRegionId(regionId)
	request.SetSecurityGroupId(securityGroupId)
	request.SetVSwitchId(vSwitchId)
	dbsC := &eci.CreateContainerGroupRequestDnsConfig{}
	request.SetDnsConfig(dbsC)
	request.SetContainerGroupName("qng-mixnet")
	ca := "NET_ADMIN"
	ssc1 := &eci.CreateContainerGroupRequestSecurityContext{}
	request.SetSecurityContext(ssc1)

	createContainerRequestContainers := make([]*eci.CreateContainerGroupRequestContainer, 1)

	createContainerRequestContainer := new(eci.CreateContainerGroupRequestContainer)
	createContainerRequestContainer.SetName("mixnet")
	createContainerRequestContainer.SetImage(qngImage)
	createContainerRequestContainer.SetCpu(2.0)
	createContainerRequestContainer.SetMemory(4.0)
	httpGet := &eci.CreateContainerGroupRequestContainerReadinessProbeHttpGet{}
	start := ""
	execC := &eci.CreateContainerGroupRequestContainerReadinessProbeExec{
		Command: []*string{&start},
	}
	port := 18160
	tcpSocket := &eci.CreateContainerGroupRequestContainerReadinessProbeTcpSocket{
		Port: &port,
	}
	readP := &eci.CreateContainerGroupRequestContainerReadinessProbe{
		HttpGet:   httpGet,
		Exec:      execC,
		TcpSocket: tcpSocket,
	}
	createContainerRequestContainer.SetReadinessProbe(readP)
	httpGetL := &eci.CreateContainerGroupRequestContainerLivenessProbeHttpGet{}
	execCL := &eci.CreateContainerGroupRequestContainerLivenessProbeExec{
		Command: []*string{&start},
	}
	tcpSocketL := &eci.CreateContainerGroupRequestContainerLivenessProbeTcpSocket{
		Port: &port,
	}
	liveP := &eci.CreateContainerGroupRequestContainerLivenessProbe{
		HttpGet:   httpGetL,
		Exec:      execCL,
		TcpSocket: tcpSocketL,
	}
	createContainerRequestContainer.SetLivenessProbe(liveP)
	capa := &eci.CreateContainerGroupRequestContainerSecurityContextCapability{
		Add: []*string{&ca},
	}
	ssc := &eci.CreateContainerGroupRequestContainerSecurityContext{
		Capability: capa,
	}
	createContainerRequestContainer.SetSecurityContext(ssc)
	qngStartVars := make([]*eci.CreateContainerGroupRequestContainerEnvironmentVar, 0)
	keys := []string{"mixnet", "rpclisten", "modules", "evmenv", "debuglevel",
		"circuit", "acceptnonstd", "rpcuser", "rpcpass", "port"}
	vals := []string{"true", "0.0.0.0:18131", "qitmeer",
		"--http --http.port=1234 --ws --ws.port=1235 --http.addr=0.0.0.0",
		"debug", "true", "true", "test", "test", "18160"}
	fieldRef := &eci.CreateContainerGroupRequestContainerEnvironmentVarFieldRef{}
	for i := 0; i < len(keys); i++ {
		qngStartVars = append(qngStartVars, &eci.CreateContainerGroupRequestContainerEnvironmentVar{
			Key:      &keys[i],
			Value:    &vals[i],
			FieldRef: fieldRef,
		})
	}
	createContainerRequestContainer.SetEnvironmentVar(qngStartVars)
	createContainerRequestContainer.SetCommand([]*string{&start})

	createContainerRequestContainers[0] = createContainerRequestContainer

	request.SetContainer(createContainerRequestContainers)
	err := tea.Validate(request)
	if err != nil {
		log.Fatalln(err)
	}
	// call api
	resp, err := client2.CreateContainerGroup(request)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(resp)
}
