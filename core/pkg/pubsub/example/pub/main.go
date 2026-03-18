/**
 * @Author:         yi
 * @Description:    sub1
 * @Version:        1.0.0
 * @Date:           2023/6/28 16:30
 */
package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"time"

	r "github.com/go-redis/redis"

	"skeyevss/core/pkg/functions"
	"skeyevss/core/pkg/pubsub"
)

func main() {
	go func() {
		_ = http.ListenAndServe("0.0.0.0:9100", nil)
	}()

	var (
		// 单机
		redisClient = r.NewClient(
			&r.Options{
				Addr:     functions.GetEnvDefault("SKEYEVSS_REDIS_PORT", ""),
				Password: functions.GetEnvDefault("SKEYEVSS_REDIS_PASSWORD", ""),
				DB:       0,
			},
		)
		// 集群
		// redisClient = r.NewClusterClient(
		// 	&r.ClusterOptions{
		// 		Addrs:        []string{functions.GetEnvDefault("SKEYEVSS_REDIS_PORT", "")},
		// 		Password:     functions.GetEnvDefault("SKEYEVSS_REDIS_PASSWORD", ""),
		// 		DialTimeout:  50 * time.Microsecond, // 设置连接超时
		// 		ReadTimeout:  50 * time.Microsecond, // 设置读取超时
		// 		WriteTimeout: 50 * time.Microsecond, // 设置写入超时
		// 	},
		// )
		client = pubsub.NewRedis(
			context.Background(),
			redisClient,
			&pubsub.Conf{},
		)
	)

	_, err := redisClient.Do("ping").String()
	if err != nil {
		panic(err)
	}
	client.PublishProc()

	for v := range time.NewTicker(time.Millisecond * 500).C {
		client.Send(
			"192.168.0."+strconv.FormatInt(v.UnixMicro()%2+1, 10),
			[]byte(strconv.FormatInt(functions.NewTimer().NowNano(), 10)),
		)
	}

	// // 存储用户和节点关联关系
	// client.SetMemberNode(111)
	// memberId, err := client.GetMemberNode(111)
	// fmt.Printf("\n memberid: %+v \n", memberId)
	// fmt.Printf("\n err: %+v \n", err)
	// return
}
