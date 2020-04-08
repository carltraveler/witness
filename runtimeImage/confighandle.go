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

const (
	prefixdir        string = "/data/"
	configRun        string = prefixdir + "config.run.json"
	configFromTenant string = prefixdir + "config.json"
	configFixed      string = "config.fixed.json"
	newcontractbash  string = "newcontract.bash"
	newcontractname  string = "contract.wasm"
	walletfixpasswd  string = "123456"
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
	configBuff, err := ioutil.ReadFile(configFromTenant)
	err = json.Unmarshal([]byte(configBuff), &configStore)
	if err != nil {
		fmt.Printf("Unmarshal configStore: %s", err)
		os.Exit(1)
	}

	fmt.Printf("configStore: %s\n", string(configBuff))

	_, err = common.AddressFromBase58(configStore.OwnerAddr)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	// config Run exist indicate just server restart
	_, err = os.Stat(configRun)
	if err != nil {
		fmt.Printf("file %s not exist\n", configRun)
		configfixedBuff, err := ioutil.ReadFile(configFixed)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}

		err = json.Unmarshal([]byte(configfixedBuff), &DefConfig)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}

		ontSdk := sdk.NewOntologySdk()
		ontSdk.NewRpcClient().SetAddress(DefConfig.OntNode)

		// construct new rust contract and compile. deploy
		buildCmd := exec.Command("bash", newcontractbash, configStore.OwnerAddr)
		err = buildCmd.Run()
		if err != nil {
			fmt.Printf("build contract err: %s", err)
			os.Exit(1)
		}

		contracthexAddr, err := DeployNewContract(ontSdk, newcontractname, walletfixpasswd, &DefConfig)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}

		DefConfig.ContracthexAddr = contracthexAddr
	} else {
		fmt.Printf("file %s exist\n", configRun)
		configfixedBuff, err := ioutil.ReadFile(configRun)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}

		err = json.Unmarshal([]byte(configfixedBuff), &DefConfig)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}

		ontSdk := sdk.NewOntologySdk()
		ontSdk.NewRpcClient().SetAddress(DefConfig.OntNode)

		if !checkContractExist(ontSdk, DefConfig.ContracthexAddr) {
			fmt.Printf("restart contracthexAddr %s not exist", DefConfig.ContracthexAddr)
			os.Exit(1)
		}
	}

	AuthAddrList := make([]string, 0)
	AuthAddrList = append(AuthAddrList, DefConfig.Authorize...)
	var duplicate bool
	for _, AuthAddr := range configStore.AuthAddr {
		duplicate = false
		for _, i := range DefConfig.Authorize {
			if AuthAddr == i {
				duplicate = true
				break
			}
		}

		if !duplicate {
			AuthAddrList = append(AuthAddrList, AuthAddr)
		}
	}

	DefConfig.Authorize = AuthAddrList

	if DefConfig.ServerPort == 0 || DefConfig.CacheTime == 0 || len(DefConfig.Walletname) == 0 || len(DefConfig.SignerAddress) == 0 || len(DefConfig.OntNode) == 0 || len(DefConfig.ContracthexAddr) == 0 || len(DefConfig.Authorize) == 0 || DefConfig.BatchNum == 0 || DefConfig.SendTxInterval == 0 || DefConfig.TryChainInterval == 0 || DefConfig.SendTxSize == 0 {
		fmt.Printf("config not set ok\n")
		os.Exit(1)
	}

	okconfig, err := json.Marshal(DefConfig)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	fmt.Printf("configRun: %s\n", string(okconfig))
	// set ok config to config.run.json
	err = ioutil.WriteFile(configRun, okconfig, 0644)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", string(okconfig))
}

func DeployNewContract(ontSdk *sdk.OntologySdk, wasmfile string, walletpassword string, newconfig *ServerConfig) (string, error) {
	code, err := ioutil.ReadFile(wasmfile)
	if err != nil {
		return "", fmt.Errorf("error in ReadFile:%s", err)
	}

	codeHash := common.ToHexString(code)
	contractAddr, err := utils.GetContractAddress(codeHash)
	if err != nil {
		return "", fmt.Errorf("GetContractAddress err: %s", err)
	}
	contracthexAddr := contractAddr.ToHexString()

	wallet, err := ontSdk.OpenWallet(newconfig.Walletname)
	if err != nil {
		return "", fmt.Errorf("error in OpenWallet:%s", err)
	}

	signer, err := wallet.GetAccountByAddress(newconfig.SignerAddress, []byte(walletpassword))
	if err != nil {
		return "", fmt.Errorf("error in GetDefaultAccount:%s", err)
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

	if !checkContractExist(ontSdk, contracthexAddr) {
		return "", fmt.Errorf("contracthexAddr %s not exist", contracthexAddr)
	}

	return contracthexAddr, nil
}

func checkContractExist(ontSdk *sdk.OntologySdk, contracthexAddr string) bool {
	checkcount := uint32(0)
	for {
		payload, err := ontSdk.GetSmartContract(contracthexAddr)
		if payload == nil || err != nil {
			if checkcount < 3 {
				fmt.Printf("GetSmartContract: %s\n", err)
				checkcount += 1
				time.Sleep(3 * time.Second)
				continue
			}

			return false
		}
		break
	}

	return true
}
