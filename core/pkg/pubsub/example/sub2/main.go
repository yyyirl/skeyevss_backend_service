/**
 * @Author:         yi
 * @Description:    sub2
 * @Version:        1.0.0
 * @Date:           2023/6/28 16:30
 */
package main

import (
	"context"
	"fmt"
	"time"

	r "github.com/go-redis/redis"

	"skeyevss/core/pkg/functions"
	"skeyevss/core/pkg/pubsub"
)

var num = 0

func main() {
	go pubsub.NewRedis(
		context.Background(),
		r.NewClient(&r.Options{
			Addr:     functions.GetEnvDefault("SKEYEVSS_REDIS_PORT", ""),
			Password: functions.GetEnvDefault("SKEYEVSS_REDIS_PASSWORD", ""),
			DB:       0,
		}),
		&pubsub.Conf{},
	).Subscribe("192.168.0.2", func(messages pubsub.RedisPublishMessageType) {
		num += len(messages)
	})

	for range time.NewTicker(time.Second * 1).C {
		fmt.Printf("\n 已接收: %+v", num)
	}
}
