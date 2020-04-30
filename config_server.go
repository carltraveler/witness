package main

import (
	"encoding/json"
	"fmt"
	//"github.com/ontio/ontology/common/log"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	MAX_REQUEST_BODY_SIZE = 1 << 10
	SUCCESS               = 0
	PARA_ERROR            = 40000
)

var ErrMap = map[int64]string{
	SUCCESS:    "SUCCESS",
	PARA_ERROR: "PARAMETER ERROR",
}

type ResponseResult struct {
	Action  string      `json:"action"`
	Error   int64       `json:"error"`
	Desc    string      `json:"desc"`
	Result  interface{} `json:"result"`
	Version string      `json:"version"`
}

func ResponseSuccess(result interface{}) *ResponseResult {
	return &ResponseResult{
		Result:  result,
		Error:   SUCCESS,
		Desc:    ErrMap[SUCCESS],
		Version: "v1",
	}
}

func ResponseFailed(errCode int64, err error, result interface{}) *ResponseResult {
	return &ResponseResult{
		Result:  result,
		Error:   errCode,
		Desc:    ErrMap[errCode] + ": " + err.Error(),
		Version: "v1",
	}
}

const (
	testUrl    string = "http://107.150.112.175:2020/addon/attestation"
	mainUrl    string = "http://52.68.40.224:2020/addon/attestation"
	mainNetype string = "main"
	testNetype string = "test"
)

type WitnessConfig struct {
	OwnerAddr string   `json:"owneraddr"`
	AuthAddr  []string `json:"authaddr"`
}

type ClientConfig struct {
	Url      string `json:"url"`
	AddOnId  string `json:"addon_id"`
	TenantId string `json:"tenant_id"`
	Wallet   string `json:"wallet"`
	Singer   string `json:"signer"`
}

type SdkResp struct {
	SdkUrl    string        `json:"sdk_url"`
	SdkConfig *ClientConfig `json:"sdk_config"`
}

type SdkInfo struct {
	AddOnId  string `json:"addon_id"`
	TenantId string `json:"tenant_id"`
	Net      string `json:"net"`
}

type SdkConfig struct {
	Config WitnessConfig `json:"config"`
	Info   SdkInfo       `json:"info"`
}

func ParseParamBody(c *gin.Context, params interface{}) error {
	bs, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bs, params)
	if err != nil {
		return err
	}

	return nil
}

func GetWitnessConfig(c *gin.Context) {
	var wconfig WitnessConfig
	c.JSON(http.StatusOK, &wconfig)
}

func PostSdkConfig(c *gin.Context) {
	var con SdkConfig
	var url string
	err := ParseParamBody(c, &con)
	if err != nil {
		c.JSON(http.StatusOK, ResponseFailed(PARA_ERROR, err, nil))
		return
	}
	fmt.Printf("sdkconfig %v\n", con)

	switch con.Info.Net {
	case testNetype:
		url = testUrl
	case mainNetype:
		url = mainUrl
	default:
		c.JSON(http.StatusOK, ResponseFailed(PARA_ERROR, nil, nil))
		return
	}

	clientConfig := &ClientConfig{
		Url:      url,
		AddOnId:  con.Info.AddOnId,
		TenantId: con.Info.TenantId,
	}
	fmt.Printf("ClientConfig %v\n", clientConfig)

	sdkresp := &SdkResp{
		SdkUrl:    "https://github.com/carltraveler/witness/blob/master/sdk/sdk.go",
		SdkConfig: clientConfig,
	}
	c.JSON(http.StatusOK, ResponseSuccess(sdkresp))
}

func NewRouter() *gin.Engine {
	root := gin.Default()
	root.GET("/config", GetWitnessConfig)
	root.POST("/config", PostSdkConfig)

	return root
}

func main() {
	r := NewRouter()
	r.Run(":8080")
}
