package rcache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// type slog interface {
// 	SetOutput(out io.Writer)
// 	Print(args ...interface{})
// 	Printf(format string, args ...interface{})
// 	Println(args ...interface{})
// 	Info(args ...interface{})
// 	Infof(format string, args ...interface{})
// 	Infoln(args ...interface{})
// 	Warn(args ...interface{})
// 	Warnf(format string, args ...interface{})
// 	Warnln(args ...interface{})
// 	Error(args ...interface{})
// 	Errorf(format string, args ...interface{})
// 	Errorln(args ...interface{})
// }

func New(redisURL string) *Cache {
	host := "localhost"
	port := 6379
	opts := &redis.Options{
		Password: "", // no password set
		// DB:       0,  // use default DB
	}

	if redisURL == "" {
		host := os.Getenv("REDISHOST")
		i, err := strconv.Atoi(os.Getenv("REDISPORT") + "")
		if err == nil {
			port = i
		}
		slog.Info("trying cache with defaults", "host", host, " port ", port)
	} else {
		opts = &redis.Options{}
		u, err := url.Parse(redisURL)
		if err != nil {
			slog.Error("error", "err", err)
			return nil
		}
		host = u.Host
		if strings.Contains(host, ":") {
			host = strings.Split(host, ":")[0]
		}
		p, _ := u.User.Password()
		opts.Password = p
		// opts.User = u.User.Username()
		port, err = strconv.Atoi(u.Port())
		if err != nil {
			slog.Error("cache couldn't parse port")
			return nil
		}
	}
	slog.Info("trying cache with redis url", "host=", host, " port= ", port)

	opts.Addr = fmt.Sprintf("%s:%d", host, port)
	client := redis.NewClient(opts)

	pong := client.Conn().Ping(context.Background())
	if err := pong.Err(); err != nil {
		slog.Error("ping err ", "err", err)
		return nil
	}

	slog.Info("cache running", pong)
	return &Cache{
		Client: client,
	}
}

type Cache struct {
	Client *redis.Client
}

func (cache *Cache) Get(ctx context.Context, key string) (string, error) {
	cmd := cache.Client.Get(ctx, key)
	err := cmd.Err()
	if err != nil {
		return "", err
	}
	val := cmd.Val()
	if val == "" {
		return "", errors.New("no value for [" + key + "]")
	}
	return val, err
}

func (cache *Cache) Del(ctx context.Context, key string) error {
	cmd := cache.Client.Expire(ctx, key, 1)
	err := cmd.Err()
	return err
}

func (cache *Cache) Expire(ctx context.Context, key string) error {
	err := cache.Del(ctx, key)
	return err
}

func (cache *Cache) GetBytes(ctx context.Context, key string) ([]byte, error) { // Encourages BLOATED interfaces
	val, err := cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func (cache *Cache) Set(ctx context.Context, key string, value string, duration time.Duration) error {
	secs := int64(duration / time.Second)
	cmd := cache.Client.SetEx(ctx, key, value, duration)
	err := cmd.Err()
	if err == nil {
		return errors.New("Not found")
	}
	if strings.Contains(err.Error(), "invalid expire time in set") {
		return errors.New("Invalid expire timeout in seconds [" + strconv.Itoa(int(secs)) + "]")
	}
	return err
}

func (cache *Cache) SetBytes(ctx context.Context, key string, value []byte, duration time.Duration) error { // Encourages BLOATED interfaces
	result := string(value[:])
	err := cache.Set(ctx, key, result, duration)
	return err
}

func (cache *Cache) FlushDB(ctx context.Context) error {
	cmd := cache.Client.FlushDB(ctx)

	return cmd.Err()
}
