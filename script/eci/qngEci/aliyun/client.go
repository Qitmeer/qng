package aliyun

import (
	"eci/config"
	"encoding/hex"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/eci"
	"log"
	"time"
)

type AliyunECI struct {
	params *config.Config
	client *eci.Client
}

func (this *AliyunECI) Init() {
	c := sdk.NewConfig()
	c.EnableAsync = this.params.EnableAsync             // Asynchronous task switch
	c.GoRoutinePoolSize = this.params.GoRoutinePoolSize // Number of goroutines
	c.MaxTaskQueueSize = this.params.MaxTaskQueueSize   // Maximum number of tasks for a single goroutine
	c.Timeout = time.Duration(this.params.Timeout) * time.Second
	credential := credentials.NewAccessKeyCredential(this.params.AccessKey, this.params.SecretKey)
	var err error
	this.client, err = eci.NewClientWithOptions(this.params.RegionId, c, credential)
	if err != nil {
		log.Fatalln(err)
		return
	}
}

func (this *AliyunECI) CreateContainer() {
	for i := 0; i < this.params.DockerContainerCount; i++ {
		request := CreateContainerGroupRequest()
		request.AutoCreateEip = requests.NewBoolean(this.params.AutoCreateEip)
		request.EipBandwidth = requests.NewInteger(this.params.EipBandwidth)
		containerGroupName := fmt.Sprintf("%s-%d", this.params.ContainerName, i)
		request.ContainerGroupName = containerGroupName
		request.RegionId = this.params.RegionId
		request.ZoneId = this.params.ZoneId
		request.Cpu = requests.NewFloat(this.params.CpuCores)
		request.SecurityGroupId = this.params.SecurityGroupId
		request.VSwitchId = this.params.VSwitchId
		if this.params.ExiprePeriod > 0 {
			request.ActiveDeadlineSeconds = requests.NewInteger(this.params.ExiprePeriod)
		}
		dockerContainerName := fmt.Sprintf("%s%d", this.params.DataDirPrefix, i)
		dn := hex.EncodeToString([]byte(dockerContainerName))
		request.Volume = &[]eci.CreateContainerGroupVolume{
			{
				Name: dn,
				Type: this.params.VolumeType,
				NFSVolume: eci.CreateContainerGroupNFSVolume{
					Server: this.params.NfsServer,
					Path:   config.PATH_SEPERATE + dockerContainerName,
				},
			},
		}
		createContainerRequestContainers := make([]eci.CreateContainerGroupContainer, 0)
		createContainerRequestContainers = append(createContainerRequestContainers, eci.CreateContainerGroupContainer{
			Name:   dn,
			Image:  this.params.QngImage,
			Cpu:    requests.NewFloat(this.params.CpuCores),
			Memory: requests.NewFloat(this.params.MemCores),
			VolumeMount: &[]eci.CreateContainerGroupVolumeMount{
				{
					Name:      dn,
					MountPath: this.params.DockerDataDir,
				},
			},
			Command: this.params.DockerExecCommand,
			Arg:     this.params.DockerExecArgs,
		})
		request.Container = &createContainerRequestContainers
		err := requests.InitParams(request.CreateContainerGroupRequest)
		if err != nil {
			log.Fatalln(err)
		}
		request.QueryParams = request.CreateContainerGroupRequest.GetQueryParams()
		response := eci.CreateCreateContainerGroupResponse()
		err = this.client.DoAction(request, response)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println("CreateContainerGroup: ", containerGroupName, " ,ContainerGroupId: ", response.ContainerGroupId)
	}
}

func (this *AliyunECI) DeleteContainer(containerId interface{}) {
	cid, ok := containerId.(string)
	if !ok {
		log.Fatalln("containerId need string")
	}
	// init request
	request := eci.CreateDeleteContainerGroupRequest()
	request.ContainerGroupId = cid
	request.RegionId = this.params.RegionId
	response := eci.CreateDeleteContainerGroupResponse()
	err := this.client.DoAction(request, response)
	if err != nil {
		log.Fatalln(err)
	}
	// call api
	fmt.Println("DeleteContainerGroup: ", containerId)
}

func (this *AliyunECI) AllContainers(containerIds interface{}, result interface{}) {
	// init request
	request := eci.CreateDescribeContainerGroupsRequest()
	request.ContainerGroupIds = containerIds.(string)
	request.RegionId = this.params.RegionId
	request.Limit = requests.NewInteger(100)
	response := eci.CreateDescribeContainerGroupsResponse()
	err := this.client.DoAction(request, response)
	if err != nil {
		log.Fatalln(err)
	}
	// call api
	for i, v := range response.ContainerGroups {
		fmt.Println(i, "ContainerGroupId", v.ContainerGroupId, "status", v.Status)
	}
}

func (this *AliyunECI) RestartContainers(containerId interface{}) {
	cid, ok := containerId.(string)
	if !ok {
		log.Fatalln("containerId need string")
	}
	request := eci.CreateRestartContainerGroupRequest()
	request.ContainerGroupId = cid
	request.RegionId = this.params.RegionId
	response := eci.CreateRestartContainerGroupResponse()
	err := this.client.DoAction(request, response)
	if err != nil {
		log.Fatalln(err)
	}
	// call api
	fmt.Println("RestartContainerGroup: ", containerId)
}

func (this *AliyunECI) SetConfig(conf *config.Config) {
	this.params = conf
}
