package cache

import (
	goredis "github.com/go-redis/redis/v8"
	"github.com/gomodule/redigo/redis"
	rejson "github.com/nitishm/go-rejson/v4"
)

func CacheSet()