package main

import (
	// "fmt"
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"log"
	"strings"
)

type resqueJob struct {
	Class string      `json:"class"`
	Args  interface{} `json:"args"`
}

type redisConn struct {
	redis.Conn
}

func writeToRedis(messages chan MessageDelivery) {
	conn, err := redisConnect()
	if err != nil {
		log.Fatalf("%v", conn)
	}
	log.Printf("Connected to redis")
	for {
		log.Println("Waiting for messages")
		message := <-messages
		payload := buildPayload(message)
		queueParts := []string{"queue", message.Mapping.WorkerQueue}
		queue := strings.Join(queueParts, ":")
		log.Println(queue)
		conn.Do("RPUSH", queue, payload)
	}
}

func buildPayload(message MessageDelivery) []byte {
	log.Printf("About to write to redis: %q", message.Body)

	var body interface{}
	json.Unmarshal(message.Body, &body)

	payload := resqueJob{message.Mapping.WorkerClass, body}
	payloadJSON, _ := json.Marshal(payload)
	log.Printf("%q", payloadJSON)
	return payloadJSON
}

func redisConnect() (*redisConn, error) {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		return nil, err
	}
	return &redisConn{Conn: conn}, nil
}
