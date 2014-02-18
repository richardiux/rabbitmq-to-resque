package main

import (
	"fmt"
	"github.com/streadway/amqp"
	"log"
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	tag     string
	done    chan error
}

func newConsumerWithReconnect(rabbitConfig RabbitConfiguration, mapping Mapping, messageChan chan MessageDelivery) error {
	consumer, err := newConsumer(rabbitConfig, mapping, messageChan)
	go func() {
		<-consumer.done
		newConsumerWithReconnect(rabbitConfig, mapping, messageChan)
	}()
	return err
}

func newConsumer(rabbitConfig RabbitConfiguration, mapping Mapping, messageChan chan MessageDelivery) (*Consumer, error) {
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

	queueArguments := amqp.Table{"x-ha-policy": "all"}
	q, err := c.channel.QueueDeclare(
		mapping.Queue,  // name
		false,          // durable
		false,          // delete when usused
		false,          // exclusive
		false,          // noWait
		queueArguments, // arguments
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

	go handle(deliveries, mapping, messageChan, c.done)
	return c, nil
}

func handle(deliveries <-chan amqp.Delivery, mapping Mapping, messageChan chan MessageDelivery, done chan error) {
	for d := range deliveries {
		log.Printf(
			"got %dB delivery: [%v] %q",
			len(d.Body),
			d.DeliveryTag,
			d.Body,
		)

		messageChan <- MessageDelivery{mapping, d.Body}
	}

	log.Printf("handle: deliveries channel closed")
	done <- nil
}
