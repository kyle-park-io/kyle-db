package client

import (
	"log"
	"net"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

var (
	RedisClient *redis.Client
	TCPClient   net.Conn
)

func InitRedisClient(port string) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:" + port,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return rdb
}

func InitTCPClient(port string) net.Conn {
	conn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		panic(err)
	}

	return conn
}

// TODO: Implement client restart process
func Init() {
	port := viper.GetString("port.redis")
	if port == "" {
		log.Fatalln("Check env: port.redis")
	}

	RedisClient = InitRedisClient(port)
	TCPClient = InitTCPClient(port)
}
