package common

import (
	"github.com/go-errors/errors"
	"sync"
	"time"
)
import "github.com/tkanos/gonfig"

var once sync.Once

const StellarImmediateOperationTimeoutSec = 60
const StellarImmediateOperationBaseFee = 200

type jsonCnfiguration struct {
	Port              int
	StellarSeed		  string
	JaegerUrl		  string
	JaegerServiceName string
	AutoFlushPeriod	  string
	MaxConcurrency	  int
	TransactionValidityPeriodSec int64
}

type configuration struct {
	Port              int
	StellarSeed		  string
	JaegerUrl		  string
	JaegerServiceName string
	AutoFlushPeriod	  time.Duration
	MaxConcurrency	  int
	TransactionValidityPeriodSec int64
}



var (
	instance configuration
)

func ParseConfiguration(configFile string) (configuration,error) {

	rawConfig := jsonCnfiguration{}

	err := gonfig.GetConf(configFile, &rawConfig)

	instance = configuration{
		Port:              rawConfig.Port,
		StellarSeed:       rawConfig.StellarSeed,
		JaegerUrl:         rawConfig.JaegerUrl,
		JaegerServiceName: rawConfig.JaegerServiceName,
		MaxConcurrency:    rawConfig.MaxConcurrency,
		TransactionValidityPeriodSec:  rawConfig.TransactionValidityPeriodSec,
	}

	instance.AutoFlushPeriod,err = time.ParseDuration(rawConfig.AutoFlushPeriod)

	// Apply defaults
	if instance.Port == 0 { instance.Port = 28080}
	if instance.MaxConcurrency == 0 { instance.MaxConcurrency = 10}
	if instance.TransactionValidityPeriodSec == 0 { instance.TransactionValidityPeriodSec = 21600}


	if err != nil {
		return configuration{}, errors.Errorf("Error parsing AutoFlushPeriod setting: " + err.Error())
	}

	if err != nil {
		return configuration{}, errors.Errorf("Error parsing configuration: " + err.Error())
	}

	return instance,nil
}
