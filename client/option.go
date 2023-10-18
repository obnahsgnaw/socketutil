package client

import (
	"go.uber.org/zap/zapcore"
	"time"
)

type Option func(*Client)

func Retry(interval time.Duration) Option {
	return func(client *Client) {
		client.retryInterval = interval
	}
}

func Timeout(timeout time.Duration) Option {
	return func(client *Client) {
		client.connectTimeout = timeout
	}
}

func Keepalive(interval time.Duration) Option {
	return func(client *Client) {
		client.keepAlive = interval
	}
}

func Logger(watcher func(level zapcore.Level, msg string)) Option {
	return func(client *Client) {
		if watcher == nil {
			watcher = func(level zapcore.Level, msg string) {}
		}
		client.logWatcher = watcher
	}
}

func Package(watcher func(mtp MsgType, msg string, pkg []byte)) Option {
	return func(client *Client) {
		if watcher == nil {
			watcher = func(mtp MsgType, msg string, pkg []byte) {}
		}
		client.pkgWatcher = watcher
	}
}

func Connect(handler func(index int)) Option {
	return func(client *Client) {
		client.listenConnect(handler)
	}
}

func Disconnect(handler func(index int)) Option {
	return func(client *Client) {
		client.listenDisconnect(handler)
	}
}

func Message(handler func(pkg []byte)) Option {
	return func(client *Client) {
		client.messageHandler = handler
	}
}
