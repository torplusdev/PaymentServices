package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Configuration struct {
	Seed    string
	Port    string
	TorAddressPrefix string
}

func GetConfig(filePath string) {
	file, errFile := os.Open(filePath)
	if errFile != nil {
		fmt.Println("error", errFile)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	configuration = Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("error:", err)
	}
}
