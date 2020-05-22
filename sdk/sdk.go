package main

import "fmt"

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ontio/ontology-crypto/keypair"
	sdk "github.com/ontio/ontology-go-sdk"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/common/password"
	"github.com/ontio/ontology/core/signature"
	"github.com/ontio/ontology/merkle"
)

//JsonRpc version
const JSON_RPC_VERSION = "2.0"

type ClientConfig struct {
	Url     string `json:"url"`
	AddOnId string `json:"addon_id"`
	TenatId string `json:"tenant_id"`
	Wallet  string `json:"wallet"`
	Singer  string `json:"signer"`
}

//JsonRpcRequest object in rpc
type JsonRpcRequest struct {
	Version string    `json:"jsonrpc"`
	Id      string    `json:"id"`
	Method  string    `json:"method"`
	Params  *RpcParam `json:"params"`
}

//JsonRpcResponse object response for JsonRpcRequest
type JsonRpcBatchAddResponse struct {
	Id     string      `json:"id"`
	Error  int64       `json:"error"`
	Desc   string      `json:"desc"`
	Result interface{} `json:"result"`
}

type JsonRpcVerifyResponse struct {
	Id     string       `json:"id"`
	Error  int64        `json:"error"`
	Desc   string       `json:"desc"`
	Result VerifyResult `json:"result"`
}

type JsonGetRootResponse struct {
	Id     string   `json:"id"`
	Error  int64    `json:"error"`
	Desc   string   `json:"desc"`
	Result RootSize `json:"result"`
}

type JsonGetContractResponse struct {
	Id     string `json:"id"`
	Error  int64  `json:"error"`
	Desc   string `json:"desc"`
	Result string `json:"result"`
}

type RootSize struct {
	Root string `json:"root"`
	Size uint32 `json:"size"`
}

//RpcClient for ontology rpc api
type RpcClient struct {
	qid        uint64
	addr       string
	httpClient *http.Client
}

//NewRpcClient return RpcClient instance
func NewRpcClient(addr string) *RpcClient {
	return &RpcClient{
		addr: addr,
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   5,
				DisableKeepAlives:     false, //enable keepalive
				IdleConnTimeout:       time.Second * 300,
				ResponseHeaderTimeout: time.Second * 300,
			},
			Timeout: time.Second * 300, //timeout for http response
		},
	}
}

//SetAddress set rpc server address. Simple http://localhost:20336
func (this *RpcClient) SetAddress(addr string) *RpcClient {
	this.addr = addr
	return this
}

func (this *RpcClient) GetNextQid() string {
	return fmt.Sprintf("%d", atomic.AddUint64(&this.qid, 1))
}

//sendRpcRequest send Rpc request to ontology
func (this *RpcClient) sendRpcRequest(clientConfig *ClientConfig, qid, method string, params *RpcParam) (interface{}, error) {
	rpcReq := &JsonRpcRequest{
		Version: JSON_RPC_VERSION,
		Id:      qid,
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("JsonRpcRequest json.Marsha error:%s", err)
	}

	req, err := http.NewRequest("POST", this.addr, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("addonID", clientConfig.AddOnId)
	req.Header.Set("tenantID", clientConfig.TenatId)
	log.Infof("%s, %s", clientConfig.AddOnId, clientConfig.TenatId)
	req.Header.Set("Content-Type", "application/json")

	resp, err := this.httpClient.Do(req)

	//resp, err := this.httpClient.Post(this.addr, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("http post request:%s error:%s", data, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read rpc response body error:%s", err)
	}

	if method == "batchAdd" {
		rpcRsp := &JsonRpcBatchAddResponse{}
		err = json.Unmarshal(body, rpcRsp)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal JsonRpcResponse:%s error:%s", body, err)
		}
		if rpcRsp.Error != 0 {
			return nil, fmt.Errorf("JsonRpcResponse error code:%d desc:%s result:%v", rpcRsp.Error, rpcRsp.Desc, rpcRsp.Result)
		}

		return rpcRsp.Result, nil
	} else if method == "verify" {
		rpcRsp := &JsonRpcVerifyResponse{}
		err = json.Unmarshal(body, rpcRsp)
		if rpcRsp.Error != 0 {
			return nil, fmt.Errorf("JsonRpcResponse error code:%d desc:%s", rpcRsp.Error, rpcRsp.Desc)
		}
		return &rpcRsp.Result, nil
	} else if method == "getRoot" {
		rpcRsp := &JsonGetRootResponse{}
		err = json.Unmarshal(body, rpcRsp)
		if rpcRsp.Error != 0 {
			return nil, fmt.Errorf("JsonRpcResponse error code:%d desc:%s", rpcRsp.Error, rpcRsp.Desc)
		}
		log.Infof("Root: %s, size: %d", rpcRsp.Result.Root, rpcRsp.Result.Size)

		return &rpcRsp.Result, nil
	} else if method == "GetContractAddress" {
		rpcRsp := &JsonGetContractResponse{}
		err = json.Unmarshal(body, rpcRsp)
		if rpcRsp.Error != 0 {
			return nil, fmt.Errorf("JsonRpcResponse error code:%d desc:%s", rpcRsp.Error, rpcRsp.Desc)
		}
		log.Infof("Contract: %s", rpcRsp.Result)

		return &rpcRsp.Result, nil

	}

	return nil, errors.New("error method")
}

func verifyLeaf(clientConfig *ClientConfig, client *RpcClient, leafs []common.Uint256) error {
	for i := uint32(0); i < uint32(len(leafs)); i++ {
		vargs := getVerifyArgs(leafs[i])
		res, err := client.sendRpcRequest(clientConfig, client.GetNextQid(), "verify", &vargs)
		if err != nil {
			return fmt.Errorf("verifyLeaf [%x] Failed: %s\n", leafs[i], err)
		}

		_, ok := res.(*VerifyResult)
		if !ok {
			return fmt.Errorf("verfiyLeaf failed. result error.")
		}
	}

	fmt.Printf("verify success.\n")
	return nil
}

func sendtx(clientConfig *ClientConfig) {
	testUrl := "http://127.0.0.1:32339"
	//testUrl := "http://127.0.0.1:32338"
	//testUrl := "http://127.0.0.1:8080"
	//testUrl := "https://attestation.ont.io"
	client := NewRpcClient(testUrl)
	SystemOut = false
	wg.Add(1)
	defer wg.Done()

	fmt.Printf("prepare args\n")
	for ; m < numbatch; m++ {
		if SystemOut {
			break
		}

		var leafs []common.Uint256
		leafs = GenerateLeafv(uint32(0)+N*m, N)
		addArgs := leafvToAddArgs(leafs)
		_, err := client.sendRpcRequest(clientConfig, client.GetNextQid(), "getRoot", nil)
		if err != nil {
			panic(err)
		}
		_, err = client.sendRpcRequest(clientConfig, client.GetNextQid(), "GetContractAddress", nil)
		if err != nil {
			panic(err)
		}

		if verify {
			verifyLeaf(clientConfig, client, leafs)
		} else {
			_, err := client.sendRpcRequest(clientConfig, client.GetNextQid(), "batchAdd", &addArgs)
			if err != nil {
				if k == 0 {
					log.Errorf("Add Error: %s, added num: %d\n", err, 0)
				} else {
					log.Errorf("Add Error: %s, added num: %d\n", err, k*m)
				}
			}
			k++
		}
	}

	fmt.Printf("sendDone.")

}

var (
	k         uint32 = uint32(0)
	m         uint32 = uint32(0)
	wg        sync.WaitGroup
	SystemOut bool = true
)

var (
	N        uint32 = 1
	numbatch uint32 = 1
	verify   bool   = true
)

var (
	configPath = flag.String("configPath", "./config.json", "configPath flag")
	sigDBPath  = flag.String("sigDBpath", "None", "sigdb path")
)

func main() {
	flag.Parse()
	log.Infof("configPath: %s\n", *configPath)
	var err error

	var clientConfig ClientConfig

	buffFixed, err := ioutil.ReadFile(*configPath)
	if err != nil {
		log.Errorf("init: %s", err)
		os.Exit(1)
	}

	err = json.Unmarshal([]byte(buffFixed), &clientConfig)
	if err != nil {
		log.Errorf("NewConfigServer: %s", err)
		os.Exit(1)
	}

	log.Infof("config fixed %v", &clientConfig)

	passwd, err := password.GetAccountPassword()
	if err != nil {
		log.Errorf("input password error %s", err)
		os.Exit(1)
	}

	err = InitSigner(clientConfig.Wallet, clientConfig.Singer, string(passwd))
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
		return
	}
	//testUrl := "http://127.0.0.1:8080"
	go sendtx(&clientConfig)
	fmt.Printf("use ctrl+c to stop\n")
	waitToExit()
}

func waitToExit() {
	exit := make(chan bool, 0)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for sig := range sc {
			SystemOut = true
			wg.Wait()
			fmt.Printf("added num: %d, tatal: %d\n", k*N, m*N)

			fmt.Printf("OGQ server received exit signal: %v.", sig.String())
			close(exit)
			break
		}
	}()
	<-exit
}

func hashLeaf(data []byte) common.Uint256 {
	tmp := append([]byte{0}, data...)
	return sha256.Sum256(tmp)
}

func GenerateLeafv(start uint32, N uint32) []common.Uint256 {
	sink := common.NewZeroCopySink(nil)
	leafs := make([]common.Uint256, 0)
	for i := uint32(start); i < start+N; i++ {
		sink.Reset()
		sink.WriteUint32(i)
		leafs = append(leafs, hashLeaf(sink.Bytes()))
	}

	return leafs
}

type RpcParam struct {
	PubKey   string   `json:"pubKey"`
	Sigature string   `json:"signature"`
	Hashes   []string `json:"hashes"`
}

func leafvToAddArgs(leafs []common.Uint256) RpcParam {
	leafargs := make([]string, 0, len(leafs))
	verifyData := make([]byte, 0)

	for i := range leafs {
		leafargs = append(leafargs, hex.EncodeToString(leafs[i][:]))
		verifyData = append(verifyData, leafs[i][:]...)
	}

	sigData, err := DefSigner.Sign(verifyData)
	if err != nil {
		panic(err)
	}

	addargs := RpcParam{
		PubKey:   hex.EncodeToString(keypair.SerializePublicKey(DefSigner.GetPublicKey())),
		Sigature: hex.EncodeToString(sigData),
		Hashes:   leafargs,
	}

	err = signature.Verify(DefSigner.GetPublicKey(), verifyData, sigData)
	if err != nil {
		panic(err)
	}

	return addargs
}

type VerifyResult struct {
	Root        common.Uint256   `json:"root"`
	TreeSize    uint32           `json:"size"`
	BlockHeight uint32           `json:"blockheight"`
	Index       uint32           `json:"index"`
	Proof       []common.Uint256 `json:"proof"`
}

func (self VerifyResult) MarshalJSON() ([]byte, error) {
	root := hex.EncodeToString(self.Root[:])
	proof := make([]string, 0, len(self.Proof))
	for i := range self.Proof {
		proof = append(proof, hex.EncodeToString(self.Proof[i][:]))
	}

	res := struct {
		Root        string   `json:"root"`
		TreeSize    uint32   `json:"size"`
		BlockHeight uint32   `json:"blockheight"`
		Index       uint32   `json:"index"`
		Proof       []string `json:"proof"`
	}{
		Root:        root,
		TreeSize:    self.TreeSize,
		BlockHeight: self.BlockHeight,
		Index:       self.Index,
		Proof:       proof,
	}

	return json.Marshal(res)
}

func (self *VerifyResult) UnmarshalJSON(buf []byte) error {
	res := struct {
		Root        string   `json:"root"`
		TreeSize    uint32   `json:"size"`
		BlockHeight uint32   `json:"blockheight"`
		Index       uint32   `json:"index"`
		Proof       []string `json:"proof"`
	}{}

	if len(buf) == 0 {
		return nil
	}

	json.Unmarshal(buf, &res)

	root, err := HashFromHexString(res.Root)
	if err != nil {
		return err
	}
	proof, err := convertParamsToLeafs(res.Proof)
	if err != nil {
		return err
	}

	self.Root = root
	self.TreeSize = res.TreeSize
	self.BlockHeight = res.BlockHeight
	self.Index = res.Index
	self.Proof = proof

	return nil
}

func convertParamsToLeafs(params []string) ([]common.Uint256, error) {
	leafs := make([]common.Uint256, len(params), len(params))

	for i := uint32(0); i < uint32(len(params)); i++ {
		s := params[i]
		leaf, err := HashFromHexString(s)
		if err != nil {
			return nil, err
		}
		leafs[i] = leaf
	}

	return leafs, nil
}

func getVerifyArgs(leaf common.Uint256) RpcParam {
	leafs := make([]string, 1, 1)
	leafs[0] = hex.EncodeToString(leaf[:])

	vargs := RpcParam{
		PubKey: hex.EncodeToString(keypair.SerializePublicKey(DefSigner.GetPublicKey())),
		Hashes: leafs,
	}

	return vargs
}

func HashFromHexString(s string) (common.Uint256, error) {
	hx, err := common.HexToBytes(s)
	if err != nil {
		return merkle.EMPTY_HASH, err
	}
	res, err := common.Uint256ParseFromBytes(hx)
	if err != nil {
		return merkle.EMPTY_HASH, err
	}
	return res, nil
}

var DefSigner sdk.Signer

func InitSigner(walletname string, signerAddress string, passwd string) error {
	DefSdk := sdk.NewOntologySdk()
	wallet, err := DefSdk.OpenWallet(walletname)
	if err != nil {
		return fmt.Errorf("error in OpenWallet:%s\n", err)
	}

	DefSigner, err = wallet.GetAccountByAddress(signerAddress, []byte(passwd))
	if err != nil {
		return fmt.Errorf("error in GetDefaultAccount:%s\n", err)
	}

	return nil
}
