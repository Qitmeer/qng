package eciV2

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/eci"
)

type CreateContainerGroupRequestV2 struct {
	eci.CreateContainerGroupRequest
	AutoCreateEip requests.Boolean `position:"Query" name:"AutoCreateEip"`
	EipBandwidth  requests.Integer `position:"Query" name:"EipBandwidth"`
}

func CreateCreateContainerGroupRequestV2() (request *CreateContainerGroupRequestV2) {
	request = &CreateContainerGroupRequestV2{}
	request.CreateContainerGroupRequest = *eci.CreateCreateContainerGroupRequest()
	return
}
