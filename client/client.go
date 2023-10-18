package client

import (
	"context"
	"errors"
	"go.uber.org/zap/zapcore"
	"io"
	"log"
	"net"
	"syscall"
	"time"
)

type MsgType int

const (
	Send    MsgType = 1
	Receive MsgType = 2
)

var etMsg = map[MsgType]string{
	Send:    "send",
	Receive: "receive",
}

func (e MsgType) String() string {
	return etMsg[e]
}

type Client struct {
	ctx                 context.Context
	cancel              context.CancelFunc
	host                string
	retryInterval       time.Duration
	connectTimeout      time.Duration
	conn                net.Conn
	connectIndex        int
	connectedHandler    []func(index int)
	disconnectIndex     int
	disconnectedHandler []func(index int)
	messageHandler      func(pkg []byte)
	pkgChan             chan []byte
	pkgWatcher          func(mtp MsgType, msg string, pkg []byte)
	logWatcher          func(level zapcore.Level, msg string)
	Tmp                 []byte
	keepAlive           time.Duration
	network             string
}

// New a socket client, network: tcp tcp4 tcp6 udp udp4 udp6 ...
func New(ctx context.Context, network string, host string, options ...Option) *Client {
	ctx1, cancel := context.WithCancel(ctx)
	if network == "" {
		network = "tcp"
	}
	c := &Client{
		ctx:            ctx1,
		cancel:         cancel,
		network:        network,
		host:           host,
		retryInterval:  time.Second * 3,
		connectTimeout: time.Second * 10,
		pkgChan:        make(chan []byte, 10),
		messageHandler: func(pkg []byte) {},
		pkgWatcher: func(mtp MsgType, msg string, pkg []byte) {
			log.Println(mtp.String(), len(pkg), "types pkg:", pkg)
		},
		logWatcher: func(level zapcore.Level, msg string) {
			log.Println(msg)
		},
	}
	c.With(options...)
	return c
}

func (c *Client) With(options ...Option) {
	for _, o := range options {
		o(c)
	}
}

func (c *Client) Host() string {
	return c.host
}

func (c *Client) Start() {
	c.startListen()
	c.dispatch()
	c.tryConnect()
	c.logWatcher(zapcore.InfoLevel, "client start")
}

func (c *Client) Stop() {
	c.logWatcher(zapcore.InfoLevel, "client stop")
	c.reset()
	c.cancel()
	close(c.pkgChan)
}

func (c *Client) Send(pkg []byte) error {
	if c.conn == nil {
		return errors.New("client error: not connected")
	}
	_, err := c.conn.Write(pkg)

	if err == nil {
		c.pkgWatcher(Send, "raw package", pkg)
	}

	return err
}

func (c *Client) listenConnect(h func(index int)) {
	if h != nil {
		c.connectedHandler = append(c.connectedHandler, h)
	}
}

func (c *Client) listenDisconnect(h func(index int)) {
	if h != nil {
		c.disconnectedHandler = append(c.disconnectedHandler, h)
	}
}

func (c *Client) Heartbeat(pkg []byte, interval time.Duration) {
	if interval <= 0 {
		c.heartbeat(pkg)
		return
	}
	if c.conn != nil {
		c.loopHandle(interval, func() bool {
			c.heartbeat(pkg)
			return true
		})
		return
	}
	c.listenConnect(func(index int) {
		if index == 1 {
			c.loopHandle(interval, func() bool {
				c.heartbeat(pkg)
				return true
			})
		}
	})
}

func (c *Client) heartbeat(pkg []byte) {
	if c.conn == nil {
		return
	}
	c.logWatcher(zapcore.DebugLevel, "heartbeat")
	if err := c.Send(pkg); err != nil {
		c.logWatcher(zapcore.ErrorLevel, "heartbeat failed,err="+err.Error())
	}
}

func (c *Client) loopHandle(interval time.Duration, cb func() bool) {
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if !cb() {
					return
				}
				if interval > 0 {
					time.Sleep(interval)
				}
			}
		}
	}(c.ctx)
}

func (c *Client) startListen() {
	c.logWatcher(zapcore.DebugLevel, "client listen start")
	c.loopHandle(0, func() bool {
		if c.conn == nil {
			time.Sleep(time.Millisecond * 100)
			return true
		}
		buf := [1024]byte{}
		n, err := c.conn.Read(buf[:])
		if err != nil || n == 0 {
			if errors.Is(syscall.EINVAL, err) || errors.Is(io.EOF, err) {
				time.Sleep(time.Millisecond * 100)
				c.reset()
				return true
			} else {
				time.Sleep(time.Millisecond * 100)
				return true
			}
		}
		packages := buf[:n]
		c.pkgChan <- packages
		return true
	})
}

func (c *Client) dispatch() {
	c.logWatcher(zapcore.DebugLevel, "client package dispatch start")
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case pkg := <-c.pkgChan:
				c.pkgWatcher(Receive, "raw package", pkg)
				c.messageHandler(pkg)
			}
		}
	}(c.ctx)
}

func (c *Client) tryConnect() {
	c.logWatcher(zapcore.DebugLevel, "client connect loop start")
	c.loopHandle(c.retryInterval, func() bool {
		if c.conn != nil {
			return true
		}

		if err := c.connect(); err != nil {
			c.logWatcher(zapcore.ErrorLevel, "client connect failed, err="+err.Error())
		} else {
			c.connectIndex++
			c.triggerConnected(c.connectIndex)
		}

		if c.retryInterval == 0 {
			c.logWatcher(zapcore.WarnLevel, "client connect loop stopped, no retry interval")
			return false
		}
		return true
	})
}

func (c *Client) connect() error {
	c.reset()
	dialer := net.Dialer{
		Timeout:   c.connectTimeout,
		KeepAlive: c.keepAlive,
	}
	conn, err := dialer.Dial(c.network, c.host)
	if err != nil {
		return err
	}

	c.conn = conn

	return nil
}

func (c *Client) reset() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.disconnectIndex++
		c.triggerDisconnected(c.disconnectIndex)
		c.conn = nil
	}
}

func (c *Client) triggerConnected(index int) {
	for _, h := range c.connectedHandler {
		h(index)
	}
}

func (c *Client) triggerDisconnected(index int) {
	for _, h := range c.disconnectedHandler {
		h(index)
	}
}
