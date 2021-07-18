package gconfig

import (
	"fmt"
)

const (
	defaultBrokerPort   = 50051
	defaultStargatePort = 50052
)

type Configfile struct {
	Broker struct {
		Address      string `json:"address"`
		BrokerPort   int    `json:"port"`
		StargatePort int    `json:"stargatePort"`
	} `json:"broker"`
	Worker struct {
		Name       string `json:"name"`
		DevoteCPUs int    `json:"devoteCPUs"`
	} `json:"provider"`
}

func BrokerPort() (port int) {
	return Config.Broker.BrokerPort
}

func StargateAddr() (addr string) {
	addr = fmt.Sprintf("%s:%d", Config.Broker.Address, Config.Broker.StargatePort)
	return
}

func StargatePort() (port int) {
	return Config.Broker.StargatePort
}

func BrokerAddr() (addr string) {
	return Config.Broker.Address
}

func BrokerDialAddr() (addr string) {
	addr = fmt.Sprintf("%s:%d", Config.Broker.Address, Config.Broker.BrokerPort)
	return
}
