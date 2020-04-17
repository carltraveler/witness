package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	sdk "github.com/ontio/ontology-go-sdk"
	"github.com/ontio/ontology-go-sdk/utils"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/types"
	utils2 "github.com/ontio/ontology/core/utils"
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

var (
	runPath    = flag.String("runPath", "/data/", "runPath flag")
	configPath = flag.String("configPath", "/appconfig/", "configPath flag")
	prefixdir  string
)

func constructInitTransation(ontSdk *sdk.OntologySdk, config *ServerConfig, signer *sdk.Account) (*types.MutableTransaction, error) {
	owner, err := common.AddressFromBase58(config.SignerAddress)
	if err != nil {
		return nil, err
	}
	contractAddress, err := common.AddressFromHexString(config.ContracthexAddr)
	if err != nil {
		return nil, err
	}

	gasPrice := config.GasPrice

	args := make([]interface{}, 2)
	args[0] = "set_owner"
	args[1] = owner

	return getTxWithArgs(ontSdk, args, gasPrice, contractAddress, signer)
}

func getTxWithArgs(ontSdk *sdk.OntologySdk, args []interface{}, gasPrice uint64, contractAddress common.Address, signer *sdk.Account) (*types.MutableTransaction, error) {
	tx, err := utils2.NewWasmVMInvokeTransaction(gasPrice, 8000000, contractAddress, args)
	if err != nil {
		return nil, fmt.Errorf("create tx failed: %s", err)
	}
	err = ontSdk.SignToTransaction(tx, signer)
	if err != nil {
		return nil, fmt.Errorf("signer tx failed: %s", err)
	}
	return tx, nil
}

func main() {
	flag.Parse()
	fmt.Printf("runPath : %s\n", *runPath)
	fmt.Printf("runPath : %s\n", *configPath)
	prefixdir = *runPath + "/"
	configRun := prefixdir + "config.run.json"
	configFromTenant := *configPath + "/config.json"
	configFixed := "config.fixed.json"
	newcontractbash := "newcontract.bash"
	newcontractname := "contract.wasm"
	walletfixpasswd := "123456"

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

	for _, addr := range configStore.AuthAddr {
		_, err = common.AddressFromBase58(addr)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}
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
		_, contracthexAddr, err := GetContractStringAndAddressByfile(newcontractname)
		if err != nil {
			fmt.Printf("get ContracthexAddr err: %s", err)
			os.Exit(1)
		}

		DefConfig.ContracthexAddr = contracthexAddr
		UpdateConfigRunAuth(&DefConfig, &configStore)
		err = WriteConfigRunJson(&DefConfig, configRun)
		if err != nil {
			fmt.Printf("WriteConfigRunJson err: %s", err)
			os.Exit(1)
		}

		signer, err := initSigner(ontSdk, &DefConfig, walletfixpasswd)
		if err != nil {
			fmt.Printf("initSigner err: %s", err)
			os.Exit(1)
		}

		initx, err := constructInitTransation(ontSdk, &DefConfig, signer)
		if err != nil {
			fmt.Printf("constructInitTransation failed %s", err)
			os.Exit(1)
		}

		_, err = DeployNewContract(ontSdk, newcontractname, &DefConfig, signer)
		if err != nil {
			fmt.Printf("DeployNewContract failed %s", err)
			os.Remove(configRun)
			os.Exit(1)
		}

		checkcount := uint32(0)
		for {
			_, err = ontSdk.SendTransaction(initx)
			if err != nil {
				if checkcount < 100 {
					fmt.Printf("SendTransaction init failed %s. try again.", err)
					checkcount += 1
					time.Sleep(3 * time.Second)
					continue
				}
				fmt.Printf("SendTransaction init failed %s", err)
				os.Exit(1)
			}
			ontSdk.WaitForGenerateBlock(30 * time.Second)
			break
		}

		fmt.Printf("contract deploy ok address :%s\n", contracthexAddr)
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

		if !checkContractExist(ontSdk, DefConfig.ContracthexAddr, 3) {
			fmt.Printf("restart contracthexAddr %s not exist", DefConfig.ContracthexAddr)
			os.Exit(1)
		}

		UpdateConfigRunAuth(&DefConfig, &configStore)
		err = WriteConfigRunJson(&DefConfig, configRun)
		if err != nil {
			fmt.Printf("WriteConfigRunJson err: %s", err)
			os.Exit(1)
		}
	}
}

func UpdateConfigRunAuth(DefConfig *ServerConfig, configStore *WitnessConfig) {
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
}

func WriteConfigRunJson(DefConfig *ServerConfig, configRun string) error {
	if DefConfig.ServerPort == 0 || DefConfig.CacheTime == 0 || len(DefConfig.Walletname) == 0 || len(DefConfig.SignerAddress) == 0 || len(DefConfig.OntNode) == 0 || len(DefConfig.ContracthexAddr) == 0 || len(DefConfig.Authorize) == 0 || DefConfig.BatchNum == 0 || DefConfig.SendTxInterval == 0 || DefConfig.TryChainInterval == 0 || DefConfig.SendTxSize == 0 {
		return fmt.Errorf("serverconfig not set ok\n")
	}

	okconfig, err := json.Marshal(DefConfig)
	if err != nil {
		return fmt.Errorf("serverconfig Marshal err: %s", err)
	}

	fmt.Printf("configRun: %s\n", string(okconfig))
	// set ok config to config.run.json
	err = ioutil.WriteFile(configRun, okconfig, 0644)
	if err != nil {
		return fmt.Errorf("WriteFile %s error: %s", configRun, err)
	}

	fmt.Printf("%s: \n%s\n", configRun, string(okconfig))
	return nil
}

func GetContractStringAndAddressByfile(wasmfile string) (string, string, error) {
	code, err := ioutil.ReadFile(wasmfile)
	if err != nil {
		return "", "", fmt.Errorf("error in ReadFile:%s", err)
	}

	codeHash := common.ToHexString(code)
	contractAddr, err := utils.GetContractAddress(codeHash)
	if err != nil {
		return "", "", fmt.Errorf("GetContractAddress err: %s", err)
	}
	contracthexAddr := contractAddr.ToHexString()
	return codeHash, contracthexAddr, nil
}

func initSigner(ontSdk *sdk.OntologySdk, newconfig *ServerConfig, walletpassword string) (*sdk.Account, error) {
	wallet, err := ontSdk.OpenWallet(newconfig.Walletname)
	if err != nil {
		return nil, fmt.Errorf("error in OpenWallet:%s", err)
	}

	signer, err := wallet.GetAccountByAddress(newconfig.SignerAddress, []byte(walletpassword))
	if err != nil {
		return nil, fmt.Errorf("error in GetDefaultAccount:%s", err)
	}

	return signer, nil
}

func DeployNewContract(ontSdk *sdk.OntologySdk, wasmfile string, newconfig *ServerConfig, signer *sdk.Account) (string, error) {
	codeHash, contracthexAddr, err := GetContractStringAndAddressByfile(wasmfile)
	if checkContractExist(ontSdk, contracthexAddr, 3) {
		return "", fmt.Errorf("contracthexAddr %s already exist. change another Owner", contracthexAddr)
	}

	fmt.Printf("start DeployNewContract: %s", contracthexAddr)
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

	ontSdk.WaitForGenerateBlock(500 * time.Second)

	if !checkContractExist(ontSdk, contracthexAddr, 100) {
		return "", fmt.Errorf("contracthexAddr %s not exist", contracthexAddr)
	}

	return contracthexAddr, nil
}

func checkContractExist(ontSdk *sdk.OntologySdk, contracthexAddr string, n uint32) bool {
	checkcount := uint32(0)
	for {
		payload, err := ontSdk.GetSmartContract(contracthexAddr)
		if payload == nil || err != nil {
			if checkcount < n {
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
