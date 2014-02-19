package main

import (
	"encoding/json"
	"fmt"
	// "github.com/streadway/amqp"
	"io/ioutil"
	// "log"
)

type Mapping struct {
	Exchange    string
	Queue       string
	RoutingKey  string
	WorkerQueue string
	WorkerClass string
}

type configuration struct {
	Mappings []Mapping
	Rabbitmq RabbitConfiguration
}

type RabbitConfiguration struct {
	Url string
}

func loadConfig(configPath string) (*configuration, error) {
	configFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("configuration error: %v\n", err)
	}

	var config *configuration
	json.Unmarshal(configFile, &config)
	return config, nil
}
