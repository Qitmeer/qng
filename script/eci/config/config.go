package config

import (
	"eci/types"
	"fmt"
	"github.com/Unknwon/goconfig"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/utils"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AccessKey            string        `default:""`
	SecretKey            string        `default:""`
	RegionId             string        `default:"cn-hongkong"`
	ZoneId               string        `default:"cn-hongkong"`
	SecurityGroupId      string        `default:""`
	NfsServer            string        `default:""`
	VSwitchId            string        `default:""`
	QngImage             string        `default:""`
	DockerContainerCount int           `default:"registry-vpc.cn-hongkong.aliyuncs.com/qng/qng-mixnet:mixnet"`
	Endpoint             string        `default:"eci.aliyuncs.com"`
	CpuCores             float64       `default:"2"`
	MemCores             float64       `default:"4"`
	DataDirPrefix        string        `default:"qngmixnet-"`
	DockerDataDir        string        `default:"/qng/data"`
	DockerExecCommand    []string      `default:""`
	ContainerName        string        `default:"qng-mixnet"`
	DockerExecArgs       []string      `default:""`
	EnableAsync          bool          `default:"true"`
	GoRoutinePoolSize    int           `default:"10"`
	MaxTaskQueueSize     int           `default:"20"`
	Timeout              int64         `default:"10"`
	ExiprePeriod         int           `default:"1800"`
	VolumeType           string        `default:"NFSVolume"`
	AutoCreateEip        bool          `default:"true"`
	EipBandwidth         int           `default:"5"`
	EciType              types.EciType `default:"0"`
}

var Params *Config

func NewConfig(filename *string) *Config {
	if Params != nil {
		return Params
	}
	var cfg *goconfig.ConfigFile
	config, err := goconfig.LoadConfigFile(*filename)
	if err != nil {
		fmt.Println("get config file error:", err.Error())
		os.Exit(-1)
	}
	Params = new(Config)
	utils.InitStructWithDefaultTag(Params)
	cfg = config
	// ================================ load config ================================
	LoadBaseConfig(cfg)
	LoadAliyunConfig(cfg)
	return Params
}
func LoadBaseConfig(cfg *goconfig.ConfigFile) {
	typName, _ := cfg.GetValue("base_conf", "eci_type")
	Params.EciType = types.GetEciType(typName)
}

func LoadAliyunConfig(cfg *goconfig.ConfigFile) {
	var err error
	Params.AccessKey, _ = cfg.GetValue("aliyun_conf", "access_key")
	Params.SecretKey, _ = cfg.GetValue("aliyun_conf", "secret_key")
	Params.RegionId, _ = cfg.GetValue("aliyun_conf", "region_id")
	Params.ZoneId, _ = cfg.GetValue("aliyun_conf", "zone_id")
	Params.SecurityGroupId, _ = cfg.GetValue("aliyun_conf", "security_group_id")
	Params.VSwitchId, _ = cfg.GetValue("aliyun_conf", "v_switch_id")
	Params.QngImage, _ = cfg.GetValue("aliyun_conf", "qng_image")
	Params.Endpoint, _ = cfg.GetValue("aliyun_conf", "endpoint")
	cs, _ := cfg.GetValue("aliyun_conf", "docker_container_count")
	Params.DockerContainerCount, err = strconv.Atoi(cs)
	if err != nil {
		log.Fatalln("docker_container_count need number", err)
	}
	Params.NfsServer, _ = cfg.GetValue("aliyun_conf", "nfs_server")
	Params.DataDirPrefix, _ = cfg.GetValue("aliyun_conf", "data_dir_prefix")
	Params.ContainerName, _ = cfg.GetValue("aliyun_conf", "container_name")
	Params.DockerDataDir, _ = cfg.GetValue("aliyun_conf", "docker_data_dir")
	cmds, _ := cfg.GetValue("aliyun_conf", "docker_exec_command")
	Params.DockerExecCommand = strings.Split(cmds, ",")
	ep, _ := cfg.GetValue("aliyun_conf", "expire_period")
	Params.ExiprePeriod, err = strconv.Atoi(ep)
	if err != nil {
		log.Fatalln("expire_period need number", err)
	}
	args, _ := cfg.GetValue("aliyun_conf", "docker_exec_args")

	Params.DockerExecArgs = strings.Split(args, ",")
}
