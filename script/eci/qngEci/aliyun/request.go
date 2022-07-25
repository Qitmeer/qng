package aliyun

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/eci"
)

type ContainerGroupRequest struct {
	*eci.CreateContainerGroupRequest
	AutoCreateEip requests.Boolean `position:"Query" name:"AutoCreateEip"`
	EipBandwidth  requests.Integer `position:"Query" name:"EipBandwidth"`
}

func CreateContainerGroupRequest() (request *ContainerGroupRequest) {
	request = &ContainerGroupRequest{}
	request.CreateContainerGroupRequest = eci.CreateCreateContainerGroupRequest()
	return
}
