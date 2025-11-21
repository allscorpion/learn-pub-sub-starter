package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	connString := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connString)

	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	log.Println("successfully connected to rabbitmq")

	channel, err := conn.Channel()

	if err != nil {
		log.Fatal(err)
	}

	_, _, err = pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, routing.GameLogSlug, routing.GameLogSlug+".*", pubsub.DURABLE)

	if err != nil {
		log.Fatal(err)
	}

	gamelogic.PrintServerHelp()

	for {
		input := gamelogic.GetInput()

		if len(input) == 0 {
			continue
		}

		firstWord := input[0]

		switch firstWord {
		case "pause":
			fmt.Println("sending a pause message")
			pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: true,
			})
		case "resume":
			fmt.Println("sending a resume message")
			pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: false,
			})
		case "exit":
			fmt.Println("exiting")
			return
		default:
			fmt.Println("unrecognized command")
		}
	}
}
