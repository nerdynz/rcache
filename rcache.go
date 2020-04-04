package rcache

import (
	"errors"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shomali11/xredis"
)

type logger interface {
	SetOutput(out io.Writer)
	Print(args ...interface{})
	Printf(format string, args ...interface{})
	Println(args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Infoln(args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Warnln(args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Errorln(args ...interface{})
}

func New(redisURL string, logger logger) *Cache {
	opts := &xredis.Options{
		Host:     "localhost",
		Port:     6379,
		Password: "", // no password set
		// DB:       0,  // use default DB
	}

	if redisURL == "" {
		opts.Host = os.Getenv("REDISHOST")
		i, err := strconv.Atoi(os.Getenv("REDISPORT") + "")
		if err == nil {
			opts.Port = i
		}
	} else {
		opts = &xredis.Options{}
		u, err := url.Parse(redisURL)
		if err != nil {
			logger.Error(err)
			return nil
		}
		opts.Host = u.Host
		if strings.Contains(opts.Host, ":") {
			opts.Host = strings.Split(opts.Host, ":")[0]
		}
		p, _ := u.User.Password()
		opts.Password = p
		// opts.User = u.User.Username()
		port, err := strconv.Atoi(u.Port())
		if err != nil {
			logger.Error("cache couldn't parse port")
			return nil
		}
		opts.Port = port
	}

	logger.Info("trying cache", opts)
	client := xredis.SetupClient(opts)
	pong, err := client.Ping()
	if err != nil {
		logger.Error(err)
		return nil
	}

	logger.Info("cache running", pong)
	return &Cache{
		Client: client,
	}
}

type Cache struct {
	Client *xredis.Client
}

func (cache *Cache) Get(key string) (string, error) {
	val, ok, err := cache.Client.Get(key)
	if val == "" {
		return "", errors.New("no value for [" + key + "]")
	}
	if !ok && err == nil {
		return "", errors.New("Not found")
	}
	return val, err
}

func (cache *Cache) Del(key string) error {
	_, err := cache.Client.Expire(key, 1)
	return err
}

func (cache *Cache) Expire(key string) error {
	err := cache.Del(key)
	return err
}

func (cache *Cache) GetBytes(key string) ([]byte, error) { // Encourages BLOATED interfaces
	val, err := cache.Get(key)
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func (cache *Cache) Set(key string, value string, duration time.Duration) error {
	secs := int64(duration / time.Second)
	ok, err := cache.Client.SetEx(key, value, secs)
	if !ok {
		if err == nil {
			return errors.New("Not found")
		}
		if strings.Contains(err.Error(), "invalid expire time in set") {
			return errors.New("Invalid expire timeout in seconds [" + strconv.Itoa(int(secs)) + "]")
		}
	}
	return err
}

func (cache *Cache) SetBytes(key string, value []byte, duration time.Duration) error { // Encourages BLOATED interfaces
	result := string(value[:])
	err := cache.Set(key, result, duration)
	return err
}

func (cache *Cache) FlushDB() error {
	return cache.Client.FlushDb()
}
