package common

import (
	"github.com/go-errors/errors"
	"sync"
	"time"
)
import "github.com/tkanos/gonfig"

var once sync.Once

type jsonCnfiguration struct {
	Port              int
	StellarSeed		  string
	JaegerUrl		  string
	JaegerServiceName string
	AutoFlushPeriod	  string
	MaxConcurrency	  int
}

type configuration struct {
	Port              int
	StellarSeed		  string
	JaegerUrl		  string
	JaegerServiceName string
	AutoFlushPeriod	  time.Duration
	MaxConcurrency	  int
}



var (
	instance configuration
	TransactionTimeoutSeconds int64 = 21600
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
	}

	instance.AutoFlushPeriod,err = time.ParseDuration(rawConfig.AutoFlushPeriod)

	if err != nil {
		return configuration{}, errors.Errorf("Error parsing AutoFlushPeriod setting: " + err.Error())
	}

	if err != nil {
		return configuration{}, errors.Errorf("Error parsing configuration: " + err.Error())
	}

	return instance,nil
}
