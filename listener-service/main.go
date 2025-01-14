package main

import (
	"fmt"
	"log"
	"math"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	// try to connect to rabbitmq
	rabbitConn, err := connect()
	if err != nil {
		log.Panicln(err)
	}
	defer rabbitConn.Close()
	log.Println("Successfully connected to the RabbitMQ server!")

	// start listening for messages

	// create consumer

	// watch the queue and consume events
}

func connect() (*amqp.Connection, error) {
	var counts int64
	var backOff = 1 * time.Second
	var connection *amqp.Connection

	// do not continue until rabbit is ready
	for {
		c, err := amqp.Dial("amqp://guest:guest@localhost")
		if err != nil {
			fmt.Println(fmt.Errorf("RabbitMQ not yet ready: %w", err))
			counts++
		} else {
			connection = c
			break
		}

		if counts > 5 {
			fmt.Println(err)
			return nil, err
		}

		seconds := int64(math.Round(math.Pow(float64(counts), 2)))
		backOff = time.Duration(seconds) * time.Second
		log.Printf("Backing off for %d seconds...\n", seconds)
		time.Sleep(backOff)
	}

	return connection, nil
}
