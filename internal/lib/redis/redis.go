package redis

import (
	"time"

	"github.com/emgag/cronmutex/internal/lib/config"
	"github.com/gomodule/redigo/redis"
)

// NewRedisConn creates new redis pool
func NewRedisConn(options *config.Options) *redis.Pool {
	var dialOptions []redis.DialOption
	{
		redis.DialConnectTimeout(5 * time.Second)
		redis.DialReadTimeout(2 * time.Second)
		redis.DialWriteTimeout(2 * time.Second)
	}

	if options.Redis.Password != "" {
		dialOptions = append(dialOptions, redis.DialPassword(options.Redis.Password))
	}

	return &redis.Pool{
		MaxIdle:   1,
		MaxActive: 1,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(options.Redis.URI, dialOptions...)
		},
	}

}
