package main

import (
	"flag"
	"log"
)

type MessageDelivery struct {
	Mapping Mapping
	Body    []byte
}

func main() {
	flag.Parse()

	var configPath = flag.String("config", "./config.json", "Configuration file in JSON")
	// configPath := "./config.json"
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("%s", err)
	}

	messageChan := make(chan MessageDelivery)

	for _, mapping := range config.Mappings {
		err := newConsumerWithReconnect(config.Rabbitmq, mapping, messageChan)
		if err != nil {
			log.Fatalf("%s", err)
		}
	}

	go writeToRedis(messageChan)

	log.Printf(" [*] Waiting for logs. To exit press CTRL+C")
	select {}
}
