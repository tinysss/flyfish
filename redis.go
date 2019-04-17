package flyfish

import (
	"flyfish/conf"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/sniperHW/kendynet/util"
	"sync"
	"sync/atomic"
)

var (
	redis_once        sync.Once
	cli               *redis.Client
	redisReqCount     int32
	redisProcessQueue *util.BlockQueue
)

func pushRedis(ctx *processContext) {
	atomic.AddInt32(&redisReqCount, 1)
	Debugln("pushRedis", ctx.getUniKey())
	redisProcessQueue.Add(ctx)
}

func pushRedisNoWait(ctx *processContext) {
	atomic.AddInt32(&redisReqCount, 1)
	Debugln("pushRedisNoWait", ctx.getUniKey())
	redisProcessQueue.AddNoWait(ctx)
}

func redisRoutine(queue *util.BlockQueue) {
	redisPipeliner_ := newRedisPipeliner(conf.RedisPipelineSize)
	for {
		closed, localList := queue.Get()
		for _, v := range localList {
			ctx := v.(*processContext)
			redisPipeliner_.append(ctx)
		}
		redisPipeliner_.exec()
		if closed {
			return
		}
	}
}

func RedisInit(host string, port int, Password string) bool {
	redis_once.Do(func() {
		cli = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", host, port),
			Password: Password,
		})

		//InitScript()

		if nil != cli {
			redisProcessQueue = util.NewBlockQueueWithName(fmt.Sprintf("redis"), conf.RedisEventQueueSize)
			for i := 0; i < conf.RedisProcessPoolSize; i++ {
				go redisRoutine(redisProcessQueue)
			}
		}
	})
	return cli != nil
}
