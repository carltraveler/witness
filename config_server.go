package main

import (
	"bytes"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ontio/ontology-crypto/keypair"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func NewClient() *http.Client {
	tr := &http.Transport{ //x509: certificate signed by unknown authority
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: tr, //x509: certificate signed by unknown authority
	}
	return client
}

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
	AuthPubKey []string `json:"authpubkey"`
}

type ClientConfig struct {
	Url      string `json:"url"`
	AddOnId  string `json:"addon_id"`
	TenantId string `json:"tenant_id"`
	Wallet   string `json:"wallet"`
	Signer   string `json:"signer"`
}

type SdkResp struct {
	SdkUrl    string        `json:"sdk_url"`
	SdkConfig *ClientConfig `json:"sdk_config"`
}

type SdkInfo struct {
	AddOnId  string `json:"addon_id"`
	TenantId string `json:"tenant_id"`
	Net      string `json:"net"`
	Product  string `json:"product"`
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
	wconfig := &WitnessConfig{
		AuthPubKey: make([]string, 0),
	}
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

	if len(con.Config.AuthPubKey) == 0 {
		c.JSON(http.StatusOK, ResponseFailed(PARA_ERROR, errors.New("AuthPubKey can not empty"), nil))
		return
	}

	for _, pub := range con.Config.AuthPubKey {
		raw, err := hex.DecodeString(pub)
		if err != nil {
			c.JSON(http.StatusOK, ResponseFailed(PARA_ERROR, err, nil))
			return
		}
		_, err = keypair.DeserializePublicKey(raw)
		if err != nil {
			c.JSON(http.StatusOK, ResponseFailed(PARA_ERROR, err, nil))
			return
		}
	}

	clientConfig := &ClientConfig{
		Url:      url,
		AddOnId:  con.Info.AddOnId,
		TenantId: con.Info.TenantId,
	}
	fmt.Printf("ClientConfig %v\n", clientConfig)

	sdkresp := &SdkResp{
		SdkUrl:    "https://github.com/leej1012/witness-java-sdk",
		SdkConfig: clientConfig,
	}

	ServerBack.Req <- &con

	c.JSON(http.StatusOK, ResponseSuccess(sdkresp))
}

type CallBackServer struct {
	Req chan *SdkConfig
}

var (
	ServerBack *CallBackServer
)

func NewCallBackServer() *CallBackServer {
	return &CallBackServer{
		Req: make(chan *SdkConfig, 10),
	}
}

type CallBackReq struct {
	TenantId string `json:"tenantId"`
	Owner    string `json:"owner"`
	Net      string `json:"net"`
}

const (
	callbacktest string = "http://test.microservice.ont.io/addon-server/api/v1/app/owner/change"
	callbackmain string = "https://prod.microservice.ont.io/addon-server/api/v1/app/owner/change"
)

func (self *CallBackServer) Callback() {
	client := NewClient()
	for {
		select {
		case req := <-self.Req:
			fmt.Printf("Callback start: %v\n", *req)
			time.Sleep(10 * time.Second)
			var posturl string
			if req.Info.Product == testNetype {
				posturl = callbacktest
			} else if req.Info.Product == mainNetype {
				posturl = callbackmain
			} else {
				fmt.Printf("Product type error %s", req.Info.Product)
				break
			}

			reqC := &CallBackReq{
				TenantId: req.Info.TenantId,
				Owner:    req.Config.AuthPubKey[0],
				Net:      req.Info.Net,
			}

			data, err := json.Marshal(reqC)

			fmt.Printf("Callback Post url %s start %s\n", posturl, string(data))
			resp, err := client.Post(posturl, "application/json", bytes.NewReader(data))
			if err != nil {
				fmt.Printf("callback post err %s", err)
				break
			}
			fmt.Printf("CallBack Ok: %v\n", *req)
			defer resp.Body.Close()
		}
	}
}

func NewRouter() *gin.Engine {
	root := gin.Default()
	root.GET("/config", GetWitnessConfig)
	root.POST("/config", PostSdkConfig)

	return root
}

func main() {
	r := NewRouter()
	ServerBack = NewCallBackServer()
	go ServerBack.Callback()
	r.Run(":8080")
}
