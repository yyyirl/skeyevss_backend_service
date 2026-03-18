/**
 * @Author:         yi
 * @Description:    redis 发布订阅
 * @Version:        1.0.0
 * @Date:           2023/6/28 23:03
 */
package pubsub

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis"

	"skeyevss/core/pkg/functions"
)

// 订阅消息类型
type redisPublishMessageChanType struct {
	channel,
	message string
}

// 提交发送消息类型
type redisMessageChanType struct {
	channel  string
	messages redisMessages
}

// 发布消息类型
type RedisPublishMessageType = []string

// RedisPublish
type ps struct {
	Ctx context.Context
	// conf
	conf *Conf
	// 关闭只执行一次
	closeOnce sync.Once
	// 发布的消息
	Message chan redisPublishMessageChanType
	// 消息列表
	Messages sync.Map
	// 各个channel的最近发送时间戳
	SendTimestamps sync.Map
	// 即将发布的消息
	PublishMessages chan *redisMessageChanType
	// 退出信号
	ExitSignal chan error
	// 是否已结束队列
	IsClosed bool
	// 内部并发关闭标记
	closed int32
}

// 单机
type RedisClient struct {
	*ps

	// 是否是集群
	isCluster bool
	// redis客户端 单机
	client *redis.Client
	// 集群
	clusterClient *redis.ClusterClient
}

// 用户和节点之间联系缓存键值
const redisMemberNodeLinkedCacheKey = "node:member"

// redis execute log prefix
const redisExecuteLogPrefix = "redis execute ------ "

// redis instance 单机
func NewRedis(ctx context.Context, client *redis.Client, conf *Conf) *RedisClient {
	return &RedisClient{
		client: client,
		ps:     newPs(ctx, conf),
	}
}

// redis instance 集群
func NewRedisCluster(ctx context.Context, client *redis.ClusterClient, conf *Conf) *RedisClient {
	return &RedisClient{
		isCluster:     true,
		clusterClient: client,
		ps:            newPs(ctx, conf),
	}
}

// ps instance
func newPs(ctx context.Context, conf *Conf) *ps {
	if conf.MaxMessageCount <= 0 {
		conf.MaxMessageCount = maxMessageCount
	}
	if conf.HeartbeatInterval <= 0 {
		conf.HeartbeatInterval = heartbeatInterval
	}
	if conf.SendInterval <= 0 {
		conf.SendInterval = sendInterval
	}

	return &ps{
		Ctx:  ctx,
		conf: conf,

		Message:         make(chan redisPublishMessageChanType, 5000),
		PublishMessages: make(chan *redisMessageChanType, 100),
		ExitSignal:      make(chan error, 100),
	}
}

// 是否已关闭
func (r *RedisClient) isClosed() bool {
	return atomic.LoadInt32(&r.closed) == 1
}

// 标记为已关闭
func (r *RedisClient) markClosed() {
	r.IsClosed = true
	atomic.StoreInt32(&r.closed, 1)
}

// 推送消息
func (r *RedisClient) Send(channel string, message []byte) {
	if r.isClosed() {
		return
	}

	r.Message <- redisPublishMessageChanType{channel, string(message)}
}

// 发布消息队列节流
func (r *RedisClient) PublishProc() {
	go r.queueProc()
	go r.heartbeatProc()
	go r.publishProc()
}

// 订阅消息
func (r *RedisClient) Subscribe(channel string, completion func(messages RedisPublishMessageType)) {
	var ps = r.subscribe(channel)
	defer func() {
		_ = ps.Close()
	}()

	const (
		workerCount = 10
		bufferSize  = 100
	)

	var (
		msgCh = make(chan RedisPublishMessageType, bufferSize)
		wg    sync.WaitGroup
	)
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range msgCh {
				completion(item)
			}
		}()
	}

	defer func() {
		close(msgCh)
		wg.Wait()
	}()

	for item := range ps.Channel() {
		if item.Payload == "" {
			continue
		}

		var list RedisPublishMessageType
		if err := functions.JSONUnmarshal([]byte(item.Payload), &list); err != nil {
			functions.LogError("消息解析失败, err: %s", err)
			continue
		}

		msgCh <- list
	}
}

// 发送消息队列
func (r *RedisClient) queueProc() {
	for {
		select {
		case <-r.Ctx.Done(): // 退出
			// 关闭队列
			r.close()
			// 发送邮件
			r.sendEmail(
				"redis publish 消息队列异常结束",
				"redis发布异常结束",
				"redis发布异常结束",
			)
			return

		case val := <-r.Message: // 接收消息
			if r.isClosed() {
				return
			}

			if val.channel == "" {
				continue
			}

			// 消息组
			var (
				now         = functions.NewTimer().NowMilli()
				messageList redisMessages
			)
			if list, ok := r.Messages.Load(val.channel); ok && list != nil {
				if ml, ok := list.(redisMessages); ok {
					messageList = ml
				}
			}

			if messageList == nil {
				messageList = make(redisMessages, 0, r.conf.MaxMessageCount)
			}

			var lastSend int64
			if ts, ok := r.SendTimestamps.Load(val.channel); ok {
				if v, ok := ts.(int64); ok {
					lastSend = v
				}
			}

			if lastSend == 0 {
				lastSend = now
			}

			// 队列数据满了 || 超出指定时间未发送
			if len(messageList) >= r.conf.MaxMessageCount || now-lastSend >= int64(r.conf.SendInterval) {
				// 订阅消息
				r.PublishMessages <- &redisMessageChanType{val.channel, messageList}
				r.SendTimestamps.Store(val.channel, now)
				r.Messages.Store(val.channel, redisMessages(nil))
				messageList = nil
			}

			// 存储队列
			r.Messages.Store(val.channel, append(messageList, val.message))

		case err := <-r.ExitSignal: // 监听退出
			// 发送邮件
			r.sendEmail(
				"redis publish 消息队列异常退出",
				"redis publish 消息队列异常退出",
				err.Error(),
			)
			if r.IsClosed {
				return
			}

			// 关闭队列
			r.close()
			return
		}
	}
}

// 心跳检测清空剩余数据
func (r *RedisClient) heartbeatProc() {
	var ticker = time.NewTicker(time.Millisecond * time.Duration(r.conf.HeartbeatInterval))
	defer ticker.Stop()

	for {
		select {
		case <-r.Ctx.Done():
			return
		case <-ticker.C:
			if r.isClosed() {
				return
			}

			var now = functions.NewTimer().NowMilli()
			r.Messages.Range(func(key, value any) bool {
				if r.isClosed() {
					return false
				}

				channel, ok := key.(string)
				if !ok {
					return true
				}

				messageList, ok := value.(redisMessages)
				if !ok || len(messageList) == 0 {
					return true
				}

				r.PublishMessages <- &redisMessageChanType{channel: channel, messages: messageList}
				r.SendTimestamps.Store(channel, now)
				r.Messages.Store(channel, redisMessages(nil))

				return true
			})
		}
	}
}

// 发送redis订阅消息
func (r *RedisClient) publishProc() {
	for {
		select {
		case <-r.Ctx.Done(): // 退出
			return

		case data := <-r.PublishMessages: // 接收消息
			if r.isClosed() {
				return
			}

			if data == nil || len(data.messages) <= 0 {
				continue
			}

			var publishMessageList RedisPublishMessageType
			for _, item := range data.messages {
				if item == "" {
					continue
				}

				publishMessageList = append(publishMessageList, item)
			}

			if len(publishMessageList) <= 0 {
				continue
			}

			message, err := functions.JSONMarshal(publishMessageList)
			if err != nil {
				functions.LogError("redis publish[" + data.channel + "] 消息序列化失败")
				continue
			}

			// 发布订阅
			if _, err := r.publish(data.channel, message).Result(); err != nil {
				if r.isClosed() {
					return
				}

				r.ExitSignal <- err
				return
			}
		}
	}
}

// redis publish
func (r *RedisClient) publish(channel string, message []byte) *redis.IntCmd {
	if r.isCluster {
		return r.clusterClient.Publish(channel, message)
	}

	return r.client.Publish(channel, message)
}

// redis subscribe
func (r *RedisClient) subscribe(channel string) *redis.PubSub {
	if r.isCluster {
		return r.clusterClient.Subscribe(channel)
	}

	return r.client.Subscribe(channel)
}

// 心跳检测清空剩余数据
func (r *RedisClient) close() {
	r.closeOnce.Do(func() {
		r.markClosed()
		close(r.Message)
		close(r.PublishMessages)
		close(r.ExitSignal)
	})
}

// 发送邮件
func (r *RedisClient) sendEmail(title, subject, body string) {
	go func() {
		if err := functions.NewMail(
			r.conf.Email.Username,
			r.conf.Email.Password,
			r.conf.Email.Host,
			r.conf.Email.Port,
		).Send(
			r.conf.Email.Emails,
			title,
			subject,
			body,
		); err != nil {
			functions.LogError("邮件发送失败 err: ", err)
		}
	}()
}

// 节点类型key
func (r *RedisClient) memberNodeKey(Type, prefix string) string {
	return Type + "-" + prefix
}

// 存储用户和节点之间的关系
func (r *RedisClient) SetMemberNode(Type string, id int64) (bool, error) {
	var (
		field = strconv.FormatInt(id, 10)
		resp  *redis.BoolCmd
	)
	if r.isCluster {
		resp = r.clusterClient.HSet(r.memberNodeKey(Type, redisMemberNodeLinkedCacheKey), field, r.conf.Host)
	} else {
		resp = r.client.HSet(r.memberNodeKey(Type, redisMemberNodeLinkedCacheKey), field, r.conf.Host)
	}

	b, err := resp.Result()
	if err != nil && err != redis.Nil {
		functions.LogError(redisExecuteLogPrefix, resp.String(), "failed, err:", err)
	} else {
		functions.LogInfo(redisExecuteLogPrefix, resp.String())
	}

	return b, err
}

// 获取用户和节点之间的关系 返回节点信息
func (r *RedisClient) GetMemberNode(Type string, id int64) (string, error) {
	var (
		field = strconv.FormatInt(id, 10)
		resp  *redis.StringCmd
	)
	if r.isCluster {
		resp = r.clusterClient.HGet(r.memberNodeKey(Type, redisMemberNodeLinkedCacheKey), field)
	} else {
		resp = r.client.HGet(r.memberNodeKey(Type, redisMemberNodeLinkedCacheKey), field)
	}

	b, err := resp.Result()
	if err != nil && err != redis.Nil {
		functions.LogError(redisExecuteLogPrefix, resp.String(), " failed, err:", err)
	} else {
		functions.LogInfo(redisExecuteLogPrefix, resp.String())
	}

	return b, err
}

// 删除用户和节点之间的关系
func (r *RedisClient) DeleteMemberNode(Type string, id int64) (int64, error) {
	var (
		field = strconv.FormatInt(id, 10)
		resp  *redis.IntCmd
	)
	if r.isCluster {
		resp = r.clusterClient.HDel(r.memberNodeKey(Type, redisMemberNodeLinkedCacheKey), field)
	} else {
		resp = r.client.HDel(r.memberNodeKey(Type, redisMemberNodeLinkedCacheKey), field)
	}

	b, err := resp.Result()
	if err != nil && err != redis.Nil {
		functions.LogError(redisExecuteLogPrefix, resp.String(), "failed, err:", err)
	} else {
		functions.LogInfo(redisExecuteLogPrefix, resp.String())
	}

	return b, err
}

// 删除所有用户和节点之间的关系
func (r *RedisClient) ClearMemberNode(Type string) (int64, error) {
	var resp *redis.IntCmd
	if r.isCluster {
		resp = r.clusterClient.Del(r.memberNodeKey(Type, redisMemberNodeLinkedCacheKey))
	} else {
		resp = r.client.Del(r.memberNodeKey(Type, redisMemberNodeLinkedCacheKey))
	}

	b, err := resp.Result()
	if err != nil && err != redis.Nil {
		functions.LogError(redisExecuteLogPrefix, resp.String(), "failed, err:", err)
	} else {
		functions.LogInfo(redisExecuteLogPrefix, resp.String())
	}

	return b, err
}

// 获取用户和节点之间的关系数量
func (r *RedisClient) GetMemberNodeCount(Type string) (int64, error) {
	var resp *redis.IntCmd
	if r.isCluster {
		resp = r.clusterClient.HLen(r.memberNodeKey(Type, redisMemberNodeLinkedCacheKey))
	} else {
		resp = r.client.HLen(r.memberNodeKey(Type, redisMemberNodeLinkedCacheKey))
	}

	b, err := resp.Result()
	if err != nil && err != redis.Nil {
		functions.LogError(redisExecuteLogPrefix, resp.String(), "failed, err:", err)
	} else {
		functions.LogInfo(redisExecuteLogPrefix, resp.String())
	}

	return b, err
}
