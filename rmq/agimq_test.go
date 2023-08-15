package rmq

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestPublishMessage(t *testing.T) {
	viper.Set("rmq_address", "amqp://admin:1234@qwer@hetu.xuelangyun.com:5672/")
	agiRMQ := New()
	for i := 0; i < 10; i++ {
		testBody := struct {
			Index int64 `json:"index"`
			Time  int64 `json:"time"`
		}{
			Index: int64(i),
			Time:  time.Now().Unix(),
		}
		err := agiRMQ.Publish("test.queue", testBody)
		// err := agiRMQ.PublishWithUid(123, "tesla-test-be.delayMsg.postKey", testBody)
		fmt.Println(i, err)
		time.Sleep(1 * time.Second)
	}
}

func TestConsumerMessage(t *testing.T) {
	// viper.Set("rmq_address", "amqp://admin:admin@10.88.255.251:5672/")
	viper.Set("rmq_address", "amqp://admin:1234@qwer@hetu.xuelangyun.com:5672/")
	agiRMQ := New()
	wg := sync.WaitGroup{}
	wg.Add(1)
	agiRMQ.RMQConsumeWithGoroutine(
		"test.queue",
		"test.queue",
		false,
		4, func(body []byte) (ack bool, err error) {
			log.Println(string(body))
			log.Println("xxxxxx")
			fmt.Println("xxxxx")
			time.Sleep(time.Second)
			return true, nil
		})
	log.Println("xxxxx")
	//
	// ch := agiRMQ.consumChannel
	// q, err := ch.QueueDeclare(
	// 	"test.queue", // name
	// 	true,         // durable
	// 	false,        // delete when unused
	// 	false,        // exclusive
	// 	false,        // no-wait
	// 	nil,          // arguments
	// )
	// failOnError(err, "Failed to declare a queue")
	// msgs, err := ch.Consume(
	// 	q.Name, // queue
	// 	"",     // consumer
	// 	false,  // auto-ack
	// 	false,  // exclusive
	// 	false,  // no-local
	// 	false,  // no-wait
	// 	nil,    // args
	// )
	// failOnError(err, "Failed to register a consumer")
	//
	// forever := make(chan bool)
	//
	// go func() {
	// 	for d := range msgs {
	// 		log.Printf("Received a message: %s", d.Body)
	// 	}
	// }()
	//
	// log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	// <-forever

	wg.Wait()
}
