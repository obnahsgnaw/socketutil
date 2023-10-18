package client

import (
	client2 "github.com/obnahsgnaw/socketutil/client"
	"github.com/obnahsgnaw/socketutil/codec"
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

func Logger(watcher func(level zapcore.Level, msg string)) Option {
	return func(client *Client) {
		if watcher == nil {
			watcher = func(level zapcore.Level, msg string) {}
		}
		client.logWatcher = watcher
		client.c.With(client2.Logger(watcher))
	}
}

func ActionLogger(watcher func(action codec.Action, msg string)) Option {
	return func(client *Client) {
		if watcher == nil {
			watcher = func(action codec.Action, msg string) {}
		}
		client.actWatcher = watcher
	}
}

func PackageLogger(watcher func(mtp client2.MsgType, msg string, pkg []byte)) Option {
	return func(client *Client) {
		if watcher == nil {
			watcher = func(mtp client2.MsgType, msg string, pkg []byte) {}
		}
		client.pkgWatcher = watcher
		client.c.With(client2.Package(watcher))
	}
}
