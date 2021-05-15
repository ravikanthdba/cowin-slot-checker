package cache

import (
	"fmt"
	"log"
	"strconv"
	"time"

	goredis "github.com/go-redis/redis/v8"
	rejson "github.com/nitishm/go-rejson/v4"
)

type RedisConnection struct {
	Connection *goredis.Client
}

func CreateConnection(hostname, password string, port int) (*RedisConnection, error) {
	var redisConnection RedisConnection
	conn := goredis.NewClient(&goredis.Options{
		Addr:     hostname + ":" + strconv.Itoa(port),
		Password: password,
		DB:       0,
	})

	redisConnection.Connection = conn
	return &redisConnection, nil
}

type RedisHandler struct {
	Handler *rejson.Handler
}

func NewReJSONHandler(conn *RedisConnection) (*RedisHandler, error) {
	log.Println("Setting Redis Handler")
	start := time.Now()
	var handler RedisHandler
	handler.Handler = rejson.NewReJSONHandler()
	handler.Handler.SetGoRedisClient(conn.Connection)
	log.Println("Handler created in: ", time.Since(start))
	return &handler, nil
}

func (rh *RedisHandler) Set(key string, subkey string, records []interface{}) (bool, error) {
	log.Println("Updating Cache Started")
	start := time.Now()
	result, err := rh.Handler.JSONSet(key, subkey, &records)
	if err != nil {
		return false, err
	}
	log.Println(result)

	if result.(string) != "OK" {
		return false, fmt.Errorf("%q", "Error Writing data to Redis")
	}

	log.Println("Updating cache completed in: ", time.Since(start))
	return true, nil
}

func (rh *RedisHandler) Get(key string, subkey string) (interface{}, error) {
	result, err := rh.Handler.JSONGet(key, subkey)
	if err != nil {
		return nil, err
	}

	return result, nil
}
