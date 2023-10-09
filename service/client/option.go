package client

import (
	client2 "github.com/obnahsgnaw/socketutil/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

type Option func(*Client)

func Retry(interval time.Duration) Option {
	return func(client *Client) {
		client.c.With(client2.Retry(interval))
	}
}

func Timeout(timeout time.Duration) Option {
	return func(client *Client) {
		client.c.With(client2.Timeout(timeout))
	}
}

func Keepalive(interval time.Duration) Option {
	return func(client *Client) {
		client.c.With(client2.Keepalive(interval))
	}
}

func Connect(handler func(index int)) Option {
	return func(client *Client) {
		client.c.With(client2.Connect(handler))
	}
}

func Disconnect(handler func(index int)) Option {
	return func(client *Client) {
		client.c.With(client2.Disconnect(handler))
	}
}

func Logger(watcher func(level zapcore.Level, msg string, data ...zap.Field)) Option {
	return func(client *Client) {
		client.watcher = watcher
		client.c.With(client2.Logger(watcher))
	}
}
