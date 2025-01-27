package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const webPort = "8080"

type Config struct {
	Rabbit *amqp.Connection
}

func main() {
	// try to connect to rabbitmq
	rabbitConn, err := connect()
	if err != nil {
		log.Panicln(err)
	}
	defer rabbitConn.Close()

	app := Config{
		Rabbit: rabbitConn,
	}
	log.Printf("Starting API server on port %s\n", webPort)

	// Defube http server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	// Start the server
	err = srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func connect() (*amqp.Connection, error) {
	var counts int64
	var backOff = 1 * time.Second
	var connection *amqp.Connection

	// do not continue until rabbit is ready
	for {
		c, err := amqp.Dial("amqp://guest:guest@rabbitmq")
		if err != nil {
			fmt.Println(fmt.Errorf("RabbitMQ not yet ready: %w", err))
			counts++
		} else {
			log.Println("Successfully connected to the RabbitMQ server!")
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
