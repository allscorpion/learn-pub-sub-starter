package main

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"github.com/rabbitmq/amqp091-go"
)

func publishLog(channel *amqp091.Channel, gamelog routing.GameLog) pubsub.AckType {
	err := pubsub.PublishGob(
		channel,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%v.%v", routing.GameLogSlug, gamelog.Username),
		gamelog,
	)

	if err != nil {
		fmt.Println(err)
		return pubsub.NackRequeue
	}

	return pubsub.Ack
}
