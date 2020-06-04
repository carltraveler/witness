package main

import (
	"testing"
	"time"
)

func TestCallBack(t *testing.T) {
	s := NewCallBackServer()
	go s.Callback()
	con := SdkConfig{
		Config: WitnessConfig{
			AuthPubKey: []string{"AuthPubKeytest"},
		},
		Info: SdkInfo{
			AddOnId:  "AddOnIdtest",
			TenantId: "TenantIdtest",
			Net:      "main",
			Product:  "main",
		},
	}
	s.Req <- &con
	time.Sleep(20 * time.Second)
}
