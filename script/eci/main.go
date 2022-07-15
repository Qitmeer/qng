package main

import (
	"eci/config"
	"flag"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/eci"
	"log"
	"time"
)

const PATH_SEP = "/"

var (
	confPath = flag.String("config", "./config.conf", "./eci --config=./config.conf")
)

func main() {
	flag.Parse()
	conf := config.NewConfig(confPath)
	c := sdk.NewConfig()
	c.EnableAsync = conf.EnableAsync             // Asynchronous task switch
	c.GoRoutinePoolSize = conf.GoRoutinePoolSize // Number of goroutines
	c.MaxTaskQueueSize = conf.MaxTaskQueueSize   // Maximum number of tasks for a single goroutine
	c.Timeout = time.Duration(conf.Timeout) * time.Second
	credential := credentials.NewAccessKeyCredential(conf.AccessKey, conf.SecretKey)
	client, err := eci.NewClientWithOptions(conf.RegionId, c, credential)
	if err != nil {
		log.Fatalln(err)
		return
	}
	for i := 0; i < conf.DockerContainerCount; i++ {
		CreateContainerGroup(i, client, conf)
	}
}

func CreateContainerGroup(i int, client *eci.Client, conf *config.Config) {
	// init request
	request := eci.CreateCreateContainerGroupRequest()
	containerGroupName := fmt.Sprintf("%s-%d", conf.ContainerName, i)
	request.ContainerGroupName = containerGroupName
	request.RegionId = conf.RegionId
	request.ZoneId = conf.ZoneId
	request.Cpu = requests.NewFloat(conf.CpuCores)
	request.SecurityGroupId = conf.SecurityGroupId
	request.VSwitchId = conf.VSwitchId
	if conf.ExiprePeriod > 0 {
		request.ActiveDeadlineSeconds = requests.NewInteger(conf.ExiprePeriod)
	}
	dockerContainerName := fmt.Sprintf("%s%d", conf.DataDirPrefix, i)
	request.Volume = &[]eci.CreateContainerGroupVolume{
		{
			Name: dockerContainerName,
			Type: conf.VolumeType,
			NFSVolume: eci.CreateContainerGroupNFSVolume{
				Server: conf.NfsServer,
				Path:   PATH_SEP + dockerContainerName,
			},
		},
	}
	createContainerRequestContainers := make([]eci.CreateContainerGroupContainer, 0)
	createContainerRequestContainers = append(createContainerRequestContainers, eci.CreateContainerGroupContainer{
		Name:   dockerContainerName,
		Image:  conf.QngImage,
		Cpu:    requests.NewFloat(conf.CpuCores),
		Memory: requests.NewFloat(conf.MemCores),
		VolumeMount: &[]eci.CreateContainerGroupVolumeMount{
			{
				Name:      dockerContainerName,
				MountPath: conf.DockerDataDir,
			},
		},
		Command: conf.DockerExecCommand,
		Arg:     conf.DockerExecArgs,
	})
	request.Container = &createContainerRequestContainers
	response := eci.CreateCreateContainerGroupResponse()
	err := client.DoAction(request, response)
	if err != nil {
		log.Fatalln(err)
	}
	// call api
	fmt.Println("CreateContainerGroup: ", containerGroupName, " ,ContainerGroupId: ", response.ContainerGroupId)
}

func DeleteContainerGroup(containerId string, client *eci.Client, conf *config.Config) {
	// init request
	request := eci.CreateDeleteContainerGroupRequest()
	request.ContainerGroupId = containerId
	request.RegionId = conf.RegionId
	response := eci.CreateDeleteContainerGroupResponse()
	err := client.DoAction(request, response)
	if err != nil {
		log.Fatalln(err)
	}
	// call api
	fmt.Println("DeleteContainerGroup: ", containerId)
}

func RestartContainerGroup(containerId string, client *eci.Client, conf *config.Config) {
	// init request
	request := eci.CreateRestartContainerGroupRequest()
	request.ContainerGroupId = containerId
	request.RegionId = conf.RegionId
	response := eci.CreateRestartContainerGroupResponse()
	err := client.DoAction(request, response)
	if err != nil {
		log.Fatalln(err)
	}
	// call api
	fmt.Println("RestartContainerGroup: ", containerId)
}

func DescribeContainerGroups(containerIds string, client *eci.Client, conf *config.Config) {
	// init request
	request := eci.CreateDescribeContainerGroupsRequest()
	request.ContainerGroupIds = containerIds
	request.RegionId = conf.RegionId
	request.Limit = requests.NewInteger(100)
	response := eci.CreateDescribeContainerGroupsResponse()
	err := client.DoAction(request, response)
	if err != nil {
		log.Fatalln(err)
	}
	// call api
	for i, v := range response.ContainerGroups {
		fmt.Println(i, "ContainerGroupId", v.ContainerGroupId, "status", v.Status)
	}
}
