package main

import (
	"encoding/json"
	"github.com/ontio/ontology/common/log"
	"net/http"

	"github.com/gorilla/mux"
)

const MAX_REQUEST_BODY_SIZE = 1 << 10

type WitnessConfig struct {
	OwnerAddr string   `json:"owneraddr"`
	AuthAddr  []string `json:"authaddr"`
}

func GetWitnessConfig(w http.ResponseWriter, req *http.Request) {
	var wconfig WitnessConfig
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(wconfig)
}

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/config", GetWitnessConfig).Methods("GET")
	log.Fatal(http.ListenAndServe(":8080", router))
}
