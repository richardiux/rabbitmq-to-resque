package main

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"io/ioutil"
	"log"
)

func loadConfig() (*configuration, error) {
	configFile, err := ioutil.ReadFile("./config.json")
	if err != nil {
		return nil, fmt.Errorf("configuration error: %v\n", err)
	}

	var config *configuration
	json.Unmarshal(configFile, &config)
	return config, nil
}

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	tag     string
	done    chan error
}

func newConsumerWithReconnect(rabbitConfig RabbitConfiguration, mapping Mapping, messageChan chan amqp.Delivery) error {
	consumer, err := newConsumer(rabbitConfig, mapping, messageChan)
	go func() {
		<-consumer.done
		newConsumerWithReconnect(rabbitConfig, mapping, messageChan)
	}()
	return err
}

func newConsumer(rabbitConfig RabbitConfiguration, mapping Mapping, messageChan chan amqp.Delivery) (*Consumer, error) {
	c := &Consumer{
		conn:    nil,
		channel: nil,
		tag:     "",
		done:    make(chan error),
	}

	var err error

	c.conn, err = amqp.Dial(rabbitConfig.Url)
	if err != nil {
		return nil, fmt.Errorf("Dial: %s", err)
	}

	go func() {
		fmt.Printf("closing: %s", <-c.conn.NotifyClose(make(chan *amqp.Error)))
	}()

	c.channel, err = c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("Channel: %s", err)
	}

	if err = c.channel.ExchangeDeclare(
		mapping.Exchange, // name
		"direct",         // type
		true,             // durable
		false,            // auto-deleted
		false,            // internal
		false,            // noWait
		nil,              // arguments
	); err != nil {
		return nil, fmt.Errorf("Exchange Declare: %s", err)
	}

	q, err := c.channel.QueueDeclare(
		mapping.Queue, // name
		false,         // durable
		false,         // delete when usused
		false,         // exclusive
		false,         // noWait
		nil,           // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("Queue Declare: %s", err)
	}

	if err = c.channel.QueueBind(
		q.Name,             // queue name
		mapping.RoutingKey, // routing key
		mapping.Exchange,   // exchange
		false,
		nil,
	); err != nil {
		return nil, fmt.Errorf("Queue Bind: %s", err)
	}

	deliveries, err := c.channel.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("Queue Consume: %s", err)
	}

	go handle(deliveries, c.done)
	return c, nil
}

func handle(deliveries <-chan amqp.Delivery, done chan error) {
	for d := range deliveries {
		log.Printf(
			"got %dB delivery: [%v] %q",
			len(d.Body),
			d.DeliveryTag,
			d.Body,
		)
	}

	log.Printf("handle: deliveries channel closed")
	done <- nil
}

type Mapping struct {
	Exchange   string
	Queue      string
	RoutingKey string
}

type configuration struct {
	Mappings []Mapping
	Rabbitmq RabbitConfiguration
}

type RabbitConfiguration struct {
	Url string
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("%s", err)
	}

	messageChan := make(chan amqp.Delivery)

	for _, mapping := range config.Mappings {
		err := newConsumerWithReconnect(config.Rabbitmq, mapping, messageChan)
		if err != nil {
			log.Fatalf("%s", err)
		}
	}

	log.Printf(" [*] Waiting for logs. To exit press CTRL+C")
	select {}
}
