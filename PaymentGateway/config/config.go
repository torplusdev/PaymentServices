package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"paidpiper.com/payment-gateway/log"

	"github.com/go-errors/errors"
	"github.com/tkanos/gonfig"
)

const StellarImmediateOperationTimeoutSec = 60
const StellarImmediateOperationBaseFee = 200

type jsonCnfiguration struct {
	Port                         int
	StellarSeed                  string
	JaegerUrl                    string
	JaegerServiceName            string
	AutoFlushPeriod              Duration
	MaxConcurrency               int
	TransactionValidityPeriodSec int64
	UseTestApi                   bool
}

type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}

type NodeConfig struct {
	AutoFlushPeriod        time.Duration
	AsyncMode              bool
	AccumulateTransactions bool
}
type RootApiConfig struct {
	UseTestApi              bool
	Seed                    string
	TransactionValiditySecs int64
}
type JaegerConfig struct {
	Url         string
	ServiceName string
}
type Configuration struct {
	RootApiConfig    RootApiConfig
	Port             int
	JaegerConfig     *JaegerConfig
	MaxConcurrency   int
	TorAddressPrefix string
	NodeConfig       NodeConfig
	ResolveKey       string
}

const torAddressPrefix = "http://localhost:5817"
const asyncMode = true
const useTestApi = true
const accumulateTransactions = true
const jaegerUrl = "http://192.168.162.128:14268/api/traces"
const jaegerServiceURL = "PaymentGatewayTest"

func DefaultCfg() *Configuration {
	return &Configuration{

		JaegerConfig: &JaegerConfig{
			Url:         jaegerUrl,
			ServiceName: jaegerServiceURL,
		},

		TorAddressPrefix: torAddressPrefix,
		ResolveKey:       "torplus",
		MaxConcurrency:   10,
		RootApiConfig: RootApiConfig{
			TransactionValiditySecs: 21600,
			UseTestApi:              true,
		},

		NodeConfig: NodeConfig{
			AutoFlushPeriod:        15 * time.Minute,
			AsyncMode:              asyncMode,
			AccumulateTransactions: accumulateTransactions,
		},
	}
}

func ParseConfiguration(configFile string) (*Configuration, error) {

	rawConfig := jsonCnfiguration{}

	err := gonfig.GetConf(configFile, &rawConfig)
	if err != nil {
		log.Error("Read json config error: ", err)
		return nil, err
	}
	instance := &Configuration{
		Port: rawConfig.Port,
		RootApiConfig: RootApiConfig{
			UseTestApi:              rawConfig.UseTestApi,
			Seed:                    rawConfig.StellarSeed,
			TransactionValiditySecs: rawConfig.TransactionValidityPeriodSec,
		},
		JaegerConfig: &JaegerConfig{
			Url:         rawConfig.JaegerUrl,
			ServiceName: rawConfig.JaegerServiceName,
		},

		MaxConcurrency: rawConfig.MaxConcurrency,
		NodeConfig: NodeConfig{
			AutoFlushPeriod:        rawConfig.AutoFlushPeriod.Duration,
			AsyncMode:              asyncMode,
			AccumulateTransactions: accumulateTransactions,
		},
	}

	defCfg := DefaultCfg()
	if instance.Port == 0 {
		instance.Port = defCfg.Port
	}
	if instance.MaxConcurrency == 0 {
		instance.MaxConcurrency = defCfg.MaxConcurrency
	}
	if instance.RootApiConfig.TransactionValiditySecs == 0 {
		instance.RootApiConfig.TransactionValiditySecs = defCfg.RootApiConfig.TransactionValiditySecs
	}
	if instance.ResolveKey == "" {
		instance.ResolveKey = defCfg.ResolveKey
	}
	instance.NodeConfig.AsyncMode = asyncMode
	instance.NodeConfig.AccumulateTransactions = accumulateTransactions
	return instance, nil
}

func ParseConfig() (*Configuration, error) {
	configPath := "config.json"
	if len(os.Args) == 2 {
		configPath = os.Args[1]
		fmt.Println(configPath)
	}
	config, err := ParseConfiguration(configPath)

	if err != nil {
		log.Error("Error reading configuration file (config.json), trying cmdline params: %v", err)
		if len(os.Args) < 3 {
			log.Panic("Reading configuration file failed, and no command line parameters supplied.")
		}
		config = DefaultCfg()
		config.RootApiConfig.Seed = os.Args[1]
		config.Port, err = strconv.Atoi(os.Args[2])
		if err != nil {
			return nil, fmt.Errorf("port supplied, but couldn't be parsed: %v", err)
		}
		return config, nil
	} else {
		if len(os.Args) >= 2 {
			config.RootApiConfig.Seed = os.Args[1]
			config.Port, err = strconv.Atoi(os.Args[2])
			if err != nil {
				return nil, fmt.Errorf("port supplied, but couldn't be parsed: %v", err)
			}
		}
	}
	return config, nil
}
