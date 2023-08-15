package main

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	"github.com/streadway/amqp"

	"github.com/DoOR-Team/goutils/rmq"
)

func main() {
	//delay msg http://blog.csdn.net/u014308482/article/details/53036770

	viper.Set("rmq_address", "amqp://admin:1234@qwer@116.62.169.85:5672/daily.tesla")
	agiRMQ := rmq.New()

	agiRMQ.ConsumeWithDelivery("test", "test", false, 4,
		func(d *amqp.Delivery) (ack bool, err error) {
			// 如果无需区分 延时过来 和 正常过来 的消息执行逻辑，则无需进行d.Exchange判断
			// 以前老的handler的body参数在这里是 d.Body
			// 其他行为保持一致
			if d.Exchange == rmq.DelayExchangeName {
				fmt.Println("delay msg->", string(d.Body), " current time->", time.Now().Unix())
			} else if d.Exchange == rmq.TopicExchangeName {
				fmt.Println("msg->", string(d.Body), " current time->", time.Now().Unix())
			}
			return true, nil
		})

	fmt.Println("running")

	c := make(chan bool)
	<-c
}
