package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	data, err := json.Marshal(val)

	if err != nil {
		return err
	}

	ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        data,
	})

	return nil
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(val)

	if err != nil {
		return err
	}

	ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/gob",
		Body:        buffer.Bytes(),
	})

	return nil
}

type SimpleQueueType string

const (
	DURABLE   SimpleQueueType = "durable"
	TRANSIENT SimpleQueueType = "transient"
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	channel, err := conn.Channel()

	if err != nil {
		return &amqp.Channel{}, amqp.Queue{}, err
	}

	durable := queueType == DURABLE
	autoDelete := queueType == TRANSIENT
	exclusive := queueType == TRANSIENT
	queue, err := channel.QueueDeclare(queueName, durable, autoDelete, exclusive, false, amqp.Table{
		"x-dead-letter-exchange": "peril_dlx",
	})

	if err != nil {
		return &amqp.Channel{}, amqp.Queue{}, err
	}

	err = channel.QueueBind(queueName, key, exchange, false, nil)

	if err != nil {
		return &amqp.Channel{}, amqp.Queue{}, err
	}

	return channel, queue, nil
}

type AckType int32

const (
	Ack         AckType = 1
	NackRequeue AckType = 2
	NackDiscard AckType = 3
)

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	channel, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)

	if err != nil {
		return err
	}

	err = channel.Qos(10, 0, false)

	if err != nil {
		return err
	}

	deliveryChannels, err := channel.Consume(queue.Name, "", false, false, false, false, nil)

	if err != nil {
		return err
	}

	go func() {
		for deliveryChannel := range deliveryChannels {
			data, err := unmarshaller(deliveryChannel.Body)
			if err != nil {
				fmt.Println(err)
				return
			}
			akyType := handler(data)
			switch akyType {
			case Ack:
				fmt.Println("message acknowledged")
				deliveryChannel.Ack(false)
			case NackRequeue:
				fmt.Println("message not acknowledged and added to requeue")
				deliveryChannel.Nack(false, true)
			case NackDiscard:
				fmt.Println("message not acknowledged and discarded")
				deliveryChannel.Nack(false, false)
			}

		}
	}()

	return nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	return subscribe(conn, exchange, queueName, key, queueType, handler, func(b []byte) (T, error) {
		var data T
		if err := json.Unmarshal(b, &data); err != nil {
			var def T
			return def, err
		}
		return data, nil
	})
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	return subscribe(conn, exchange, queueName, key, queueType, handler, func(b []byte) (T, error) {
		buffer := bytes.NewBuffer(b)
		encoder := gob.NewDecoder(buffer)
		var data T
		err := encoder.Decode(&data)
		if err != nil {
			return data, err
		}
		return data, nil
	})
}
