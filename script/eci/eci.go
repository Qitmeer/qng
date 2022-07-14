package main

import (
	"fmt"
	eci "github.com/alibabacloud-go/eci-20180808/client"
	util "github.com/alibabacloud-go/tea-utils/service"
	"github.com/alibabacloud-go/tea/tea"
)

func CreateContainerGroupV2(client *eci.Client, request *eci.CreateContainerGroupRequest) (_result *eci.CreateContainerGroupResponse, _err error) {
	runtimeObject := new(util.RuntimeOptions).SetAutoretry(false).
		SetMaxIdleConns(3)
	_err = util.ValidateModel(request)
	if _err != nil {
		return _result, _err
	}
	jsonMap := tea.ToMap(request)

	// add public ip
	jsonMap["AutoCreateEip"] = true
	// default 5M
	jsonMap["EipBandwidth"] = 5

	_result = &eci.CreateContainerGroupResponse{}
	_body, _err := client.DoRequest(
		tea.String("CreateContainerGroup"),
		tea.String("HTTPS"), tea.String("POST"),
		tea.String("2018-08-08"),
		tea.String("AK"),
		nil,
		jsonMap,
		runtimeObject)
	if _err != nil {
		return _result, _err
	}
	_err = tea.Convert(_body, &_result)
	return _result, _err
}

func CreateContainerGroup(i int) {
	// init request
	request := new(eci.CreateContainerGroupRequest)
	request.SetRegionId(regionId)
	request.SetSecurityGroupId(securityGroupId)
	request.SetVSwitchId(vSwitchId)
	dbsC := &eci.CreateContainerGroupRequestDnsConfig{}
	request.SetDnsConfig(dbsC)
	request.SetContainerGroupName(fmt.Sprintf("%s-%d", containerName, i))
	ca := "NET_ADMIN"
	ssc1 := &eci.CreateContainerGroupRequestSecurityContext{}
	request.SetSecurityContext(ssc1)
	if exiprePeriod > 0 {
		request.SetActiveDeadlineSeconds(exiprePeriod)
	}

	request.SetAutoMatchImageCache(true)
	createContainerRequestContainers := make([]*eci.CreateContainerGroupRequestContainer, 0)
	volumes := []*eci.CreateContainerGroupRequestVolume{}
	volumeType := "NFSVolume"
	// nfs storage must mkdir dirs qngmixnet-0 - qngmixnet-99 first
	dockerContainerName := fmt.Sprintf("%s%d", dataDirPrefix, i)
	// first run
	// create dirs
	start := "ls"
	sPath := "/" + dockerContainerName
	nfsVolume := &eci.CreateContainerGroupRequestVolumeNFSVolume{
		Server: &nfsServer,
		Path:   &sPath,
	}
	cf := &eci.CreateContainerGroupRequestVolumeConfigFileVolumeConfigFileToPath{}
	volumes = append(volumes, &eci.CreateContainerGroupRequestVolume{
		Name:      &dockerContainerName,
		Type:      &volumeType,
		NFSVolume: nfsVolume,
		ConfigFileVolume: &eci.CreateContainerGroupRequestVolumeConfigFileVolume{
			ConfigFileToPath: []*eci.CreateContainerGroupRequestVolumeConfigFileVolumeConfigFileToPath{
				cf,
			},
		},
		EmptyDirVolume: &eci.CreateContainerGroupRequestVolumeEmptyDirVolume{},
		DiskVolume:     &eci.CreateContainerGroupRequestVolumeDiskVolume{},
		FlexVolume:     &eci.CreateContainerGroupRequestVolumeFlexVolume{},
		HostPathVolume: &eci.CreateContainerGroupRequestVolumeHostPathVolume{},
	})
	createContainerRequestContainer := new(eci.CreateContainerGroupRequestContainer)
	createContainerRequestContainer.SetName(dockerContainerName)
	createContainerRequestContainer.SetImage(qngImage)
	createContainerRequestContainer.SetCpu(cpuCores)
	createContainerRequestContainer.SetMemory(memCores)

	readonly := false
	mount := &eci.CreateContainerGroupRequestContainerVolumeMount{
		Name:      &dockerContainerName,
		MountPath: &dockerDataDir,
		ReadOnly:  &readonly,
	}
	mounts := []*eci.CreateContainerGroupRequestContainerVolumeMount{
		mount,
	}
	createContainerRequestContainer.SetVolumeMount(mounts)
	httpGet := &eci.CreateContainerGroupRequestContainerReadinessProbeHttpGet{}
	execC := &eci.CreateContainerGroupRequestContainerReadinessProbeExec{
		Command: []*string{&start},
	}
	tcpSocket := &eci.CreateContainerGroupRequestContainerReadinessProbeTcpSocket{}
	delaySec := 5
	readP := &eci.CreateContainerGroupRequestContainerReadinessProbe{
		HttpGet:             httpGet,
		Exec:                execC,
		TcpSocket:           tcpSocket,
		InitialDelaySeconds: &delaySec,
	}
	createContainerRequestContainer.SetReadinessProbe(readP)
	httpGetL := &eci.CreateContainerGroupRequestContainerLivenessProbeHttpGet{}
	execCL := &eci.CreateContainerGroupRequestContainerLivenessProbeExec{
		Command: []*string{&start},
	}
	tcpSocketL := &eci.CreateContainerGroupRequestContainerLivenessProbeTcpSocket{}
	liveP := &eci.CreateContainerGroupRequestContainerLivenessProbe{
		HttpGet:             httpGetL,
		Exec:                execCL,
		TcpSocket:           tcpSocketL,
		InitialDelaySeconds: &delaySec,
	}
	createContainerRequestContainer.SetLivenessProbe(liveP)
	capa := &eci.CreateContainerGroupRequestContainerSecurityContextCapability{
		Add: []*string{&ca},
	}
	ssc := &eci.CreateContainerGroupRequestContainerSecurityContext{
		Capability: capa,
	}
	createContainerRequestContainer.SetSecurityContext(ssc)
	keys := []*string{}
	for i := 0; i < len(dockerExecArgs); i++ {
		keys = append(keys, &dockerExecArgs[i])
	}
	createContainerRequestContainer.SetArg(keys)
	createContainerRequestContainer.SetCommand([]*string{&dockerExecCommand})
	createContainerRequestContainers = append(createContainerRequestContainers, createContainerRequestContainer)

	eks := "PATH"
	evs := "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	fr := &eci.CreateContainerGroupRequestContainerEnvironmentVarFieldRef{}
	ep := &eci.CreateContainerGroupRequestContainerEnvironmentVar{
		Key:      &eks,
		Value:    &evs,
		FieldRef: fr,
	}
	createContainerRequestContainer.SetEnvironmentVar([]*eci.CreateContainerGroupRequestContainerEnvironmentVar{ep})
	createContainerRequestContainer.SetImagePullPolicy("IfNotPresent")
	request.SetVolume(volumes)
	request.SetTerminationGracePeriodSeconds(30)
	request.SetSlsEnable(false)
	request.SetContainer(createContainerRequestContainers)

	// call api
	resp, err := CreateContainerGroupV2(client, request)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("CreateContainerGroup: ", containerName, i, " ,ContainerGroupId: ", resp.ContainerGroupId)
}

func deleteContainerGroupById(containerGroupId string) {
	deleteContainerGroupRequest := new(eci.DeleteContainerGroupRequest)
	deleteContainerGroupRequest.RegionId = &regionId
	deleteContainerGroupRequest.ContainerGroupId = &containerGroupId

	_, err := client.DeleteContainerGroup(deleteContainerGroupRequest)
	if err != nil {
		panic(err)
	}

	fmt.Println("DeleteContainerGroup ContainerGroupId :", containerGroupId)
}

func restartContainerGroupById(containerGroupId string) {
	restartContainerGroupRequest := new(eci.RestartContainerGroupRequest)
	restartContainerGroupRequest.RegionId = &regionId
	restartContainerGroupRequest.ContainerGroupId = &containerGroupId

	_, err := client.RestartContainerGroup(restartContainerGroupRequest)
	if err != nil {
		panic(err)
	}

	fmt.Println("RestartContainerGroup ContainerGroupId :", containerGroupId)
}
