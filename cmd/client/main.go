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

	username, err := gamelogic.ClientWelcome()

	if err != nil {
		log.Fatal(err)
	}

	channel, err := conn.Channel()

	if err != nil {
		log.Fatal(err)
	}

	gameState := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%v.%v", routing.PauseKey, username),
		routing.PauseKey,
		pubsub.TRANSIENT,
		handlerPause(gameState),
	)

	if err != nil {
		log.Fatal(err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%v.%v", routing.ArmyMovesPrefix, username),
		routing.ArmyMovesPrefix+".*",
		pubsub.TRANSIENT,
		handlerMove(gameState, channel),
	)

	if err != nil {
		log.Fatal(err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.DURABLE,
		handlerWarMessages(gameState, channel, username),
	)

	if err != nil {
		log.Fatal(err)
	}

	for {
		input := gamelogic.GetInput()

		if len(input) == 0 {
			continue
		}

		command := input[0]

		switch command {
		case "spawn":
			err := gameState.CommandSpawn(input)
			if err != nil {
				fmt.Println(err)
			}
		case "move":
			move, err := gameState.CommandMove(input)
			if err != nil {
				fmt.Println(err)
				continue
			}
			err = pubsub.PublishJSON(channel, routing.ExchangePerilTopic, fmt.Sprintf("%v.%v", routing.ArmyMovesPrefix, username), move)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println("move has been published")
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("spamming not currently supported")
		case "exit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("unrecognized command")
		}
	}
}
