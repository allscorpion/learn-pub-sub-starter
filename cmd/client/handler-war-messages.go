package main

import (
	"fmt"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"github.com/rabbitmq/amqp091-go"
)

func publishWarLog(channel *amqp091.Channel, username string, log string) pubsub.AkyType {
	err := pubsub.PublishGob(
		channel,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%v.%v", routing.GameLogSlug, username),
		routing.GameLog{
			CurrentTime: time.Now().UTC(),
			Message:     log,
			Username:    username,
		},
	)

	if err != nil {
		fmt.Println(err)
		return pubsub.NackRequeue
	}

	return pubsub.Ack
}

func handlerWarMessages(gs *gamelogic.GameState, channel *amqp091.Channel, username string) func(gamelogic.RecognitionOfWar) pubsub.AkyType {
	return func(rw gamelogic.RecognitionOfWar) pubsub.AkyType {
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(rw)

		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			return publishWarLog(channel, username, fmt.Sprintf("%v won a war against %v\n", winner, loser))
		case gamelogic.WarOutcomeYouWon:
			return publishWarLog(channel, username, fmt.Sprintf("%v won a war against %v\n", winner, loser))
		case gamelogic.WarOutcomeDraw:
			return publishWarLog(channel, username, fmt.Sprintf("A war between %v and %v resulted in a draw", winner, loser))
		default:
			fmt.Printf("unrecognized war outcome %v: discarding message\n", outcome)
			return pubsub.NackDiscard
		}
	}
}
