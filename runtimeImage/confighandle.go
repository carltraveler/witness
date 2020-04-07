package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	sdk "github.com/ontio/ontology-go-sdk"
	"github.com/ontio/ontology-go-sdk/utils"
	"github.com/ontio/ontology/common"
)

type ServerConfig struct {
	Walletname        string   `json:"walletname"`
	OntNode           string   `json:"ontnode"`
	SignerAddress     string   `json:"signeraddress"`
	ServerPort        int      `json:"serverport"`
	GasPrice          uint64   `json:"gasprice"`
	CacheTime         uint32   `json:"cachetime"`
	BatchNum          uint32   `json:"batchnum"`
	TryChainInterval  uint32   `json:"trychaininterval"`
	SendTxInterval    uint32   `json:"sendtxinterval"`
	SendTxSize        uint32   `json:"sendtxsize"`
	BatchAddSleepTime uint32   `json:"batchaddsleeptime"`
	ContracthexAddr   string   `json:"contracthexaddr"`
	Authorize         []string `json:"authorize"`
}

type WitnessConfig struct {
	OwnerAddr string   `json:"owneraddr"`
	AuthAddr  []string `json:"authaddr"`
}

func main() {
	var configStore WitnessConfig
	var DefConfig ServerConfig
	configBuff, err := ioutil.ReadFile("config.json")
	err = json.Unmarshal([]byte(configBuff), &configStore)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	_, err = common.AddressFromBase58(configStore.OwnerAddr)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	configfixedBuff, err := ioutil.ReadFile("config.fixed.json")
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	err = json.Unmarshal([]byte(configfixedBuff), &DefConfig)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	// construct new rust contract and compile. deploy
	buildCmd := exec.Command("bash", "newcontract.bash", configStore.OwnerAddr)
	err = buildCmd.Run()
	if err != nil {
		fmt.Printf("build contract err: %s", err)
		os.Exit(1)
	}

	contracthexAddr, err := DeployNewContract("contract.wasm", "123456", &DefConfig)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	DefConfig.Authorize = append(DefConfig.Authorize, configStore.AuthAddr...)
	DefConfig.ContracthexAddr = contracthexAddr

	if DefConfig.ServerPort == 0 || DefConfig.CacheTime == 0 || len(DefConfig.Walletname) == 0 || len(DefConfig.SignerAddress) == 0 || len(DefConfig.OntNode) == 0 || len(DefConfig.ContracthexAddr) == 0 || len(DefConfig.Authorize) == 0 || DefConfig.BatchNum == 0 || DefConfig.SendTxInterval == 0 || DefConfig.TryChainInterval == 0 || DefConfig.SendTxSize == 0 {
		fmt.Printf("config not set ok\n")
		os.Exit(1)
	}

	okconfig, err := json.Marshal(DefConfig)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	// set ok config to config.run.json
	err = ioutil.WriteFile("config.run.json", okconfig, 0644)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", string(okconfig))
}

func DeployNewContract(wasmfile string, walletpassword string, newconfig *ServerConfig) (string, error) {
	testUrl := newconfig.OntNode
	ontSdk := sdk.NewOntologySdk()
	ontSdk.NewRpcClient().SetAddress(testUrl)
	wallet, err := ontSdk.OpenWallet(newconfig.Walletname)
	if err != nil {
		return "", fmt.Errorf("error in OpenWallet:%s", err)
	}

	signer, err := wallet.GetAccountByAddress(newconfig.SignerAddress, []byte(walletpassword))
	if err != nil {
		return "", fmt.Errorf("error in GetDefaultAccount:%s", err)
	}

	code, err := ioutil.ReadFile(wasmfile)
	if err != nil {
		return "", fmt.Errorf("error in ReadFile:%s", err)
	}

	codeHash := common.ToHexString(code)
	contractAddr, err := utils.GetContractAddress(codeHash)
	if err != nil {
		return "", fmt.Errorf("GetContractAddress err: %s", err)
	}

	gasprice := newconfig.GasPrice
	deploygaslimit := uint64(200000000)
	_, err = ontSdk.WasmVM.DeployWasmVMSmartContract(
		gasprice,
		deploygaslimit,
		signer,
		codeHash,
		"witness contract",
		"1.0",
		"author",
		"email",
		"desc",
	)
	if err != nil {
		return "", fmt.Errorf("error in DeployWasmVMSmartContract:%s", err)
	}
	_, err = ontSdk.WaitForGenerateBlock(30 * time.Second)
	if err != nil {
		return "", fmt.Errorf("error in WaitForGenerateBlock:%s", err)
	}

	contracthexAddr := contractAddr.ToHexString()

	checkcount := uint32(0)
	for {
		payload, err := ontSdk.GetSmartContract(contracthexAddr)
		if payload == nil || err != nil {
			if checkcount < 3 {
				time.Sleep(3 * time.Second)
				continue
			}
			return "", fmt.Errorf("contract deploy failed %s", err)
		}
		checkcount += 1
		break
	}

	return contracthexAddr, nil
}
