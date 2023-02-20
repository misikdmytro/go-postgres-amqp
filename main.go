package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type PostgresUpdate[T any] struct {
	New T `json:"new"`
	Old T `json:"old"`
}

func (u User) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

func main() {
	const (
		queueName    = "pg"
		exchangeName = "pg"
	)

	currentEnv := os.Getenv("ENV")
	if currentEnv == "" {
		currentEnv = "local"
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.SetConfigFile(fmt.Sprintf("./config/%s.env.yaml", currentEnv))
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	rabbitMqHost := viper.GetString("RABBITMQ_HOST")
	redisHost := viper.GetString("REDIS_HOST")

	conn, err := amqp.Dial(rabbitMqHost)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-single-active-consumer": true,
		},
	)
	if err != nil {
		panic(err)
	}

	err = ch.QueueBind(
		q.Name,
		"#",
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisHost,
	})

	msgs, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					msg.Ack(false)
					return
				}

				var update PostgresUpdate[User]
				if err := json.Unmarshal(msg.Body, &update); err != nil {
					log.Println(err)
					msg.Ack(false)
					continue
				}

				log.Printf("received updated %v", update)
				if update.New.ID == 0 {
					if err := rdb.Del(ctx, fmt.Sprintf("user:%d", update.Old.ID)).Err(); err != nil {
						log.Println(err)
						msg.Ack(false)
						continue
					}
				} else {
					if err := rdb.Set(ctx, fmt.Sprintf("user:%d", update.New.ID), update.New, 0).Err(); err != nil {
						log.Println(err)
						msg.Ack(false)
						continue
					}
				}
			}
		}
	}()

	wg.Wait()
}
