package client

import (
	"context"
	"errors"
	"github.com/gorilla/websocket"
	"go.uber.org/zap/zapcore"
	"io"
	"log"
	"net"
	"sync"
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
	wsConn              *websocket.Conn
	wsLock              sync.Mutex
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
	heartbeatCancel     context.CancelFunc
	heartbeatPaused     bool
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

func (c *Client) Send(pkg []byte) (err error) {
	if c.conn == nil && c.wsConn == nil {
		return errors.New("client error: not connected")
	}
	if c.conn != nil {
		_, err = c.conn.Write(pkg)
	} else {
		c.wsLock.Lock()
		defer c.wsLock.Unlock()
		err = c.wsConn.WriteMessage(websocket.TextMessage, pkg)
	}

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

func (c *Client) HeartbeatPause() {
	c.heartbeatPaused = true
}

func (c *Client) HeartbeatContinue() {
	c.heartbeatPaused = false
}

func (c *Client) Heartbeat(pkg []byte, interval time.Duration) {
	if interval <= 0 {
		c.heartbeat(pkg)
		return
	}
	if c.conn != nil || c.wsConn != nil {
		ctx, cancel := context.WithCancel(c.ctx)
		if c.heartbeatCancel != nil {
			c.heartbeatCancel()
		}
		c.heartbeatCancel = cancel
		c.loopHandle(ctx, interval, func() bool {
			if !c.heartbeatPaused {
				c.heartbeat(pkg)
			}
			return true
		})
		return
	}
}

func (c *Client) heartbeat(pkg []byte) {
	if c.conn == nil || c.wsConn == nil {
		return
	}
	c.logWatcher(zapcore.DebugLevel, "heartbeat")
	if err := c.Send(pkg); err != nil {
		c.logWatcher(zapcore.ErrorLevel, "heartbeat failed,err="+err.Error())
	}
}

func (c *Client) loopHandle(ctx context.Context, interval time.Duration, cb func() bool) {
	go func(ctx1 context.Context) {
		for {
			select {
			case <-ctx1.Done():
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
	}(ctx)
}

func (c *Client) startListen() {
	c.logWatcher(zapcore.DebugLevel, "client listen start")
	c.loopHandle(c.ctx, 0, func() bool {
		if c.conn == nil && c.wsConn == nil {
			time.Sleep(time.Millisecond * 100)
			return true
		}
		var packages []byte
		var err error
		if c.conn != nil {
			buf := [1024]byte{}
			var n int
			n, err = c.conn.Read(buf[:])
			if err == nil && n > 0 {
				packages = buf[:n]
			}
		} else {
			_, packages, err = c.wsConn.ReadMessage()
		}
		if err != nil {
			var closeError *websocket.CloseError
			if errors.Is(syscall.EINVAL, err) || errors.Is(io.EOF, err) || errors.As(err, &closeError) {
				c.reset()
			}
			time.Sleep(time.Millisecond * 100)
			return true
		}
		if len(packages) > 0 {
			c.pkgChan <- packages
		}
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
	c.loopHandle(c.ctx, c.retryInterval, func() bool {
		if c.conn != nil || c.wsConn != nil {
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
	if c.network == "ws" || c.network == "wss" {
		conn, _, err := websocket.DefaultDialer.Dial(c.network+"://"+c.host, nil)
		if err != nil {
			return err
		}
		c.wsConn = conn
	} else {
		dialer := net.Dialer{
			Timeout:   c.connectTimeout,
			KeepAlive: c.keepAlive,
		}
		conn, err := dialer.Dial(c.network, c.host)
		if err != nil {
			return err
		}

		c.conn = conn
	}

	return nil
}

func (c *Client) reset() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.disconnectIndex++
		c.triggerDisconnected(c.disconnectIndex)
		c.conn = nil
	}
	if c.wsConn != nil {
		_ = c.wsConn.Close()
		c.disconnectIndex++
		c.triggerDisconnected(c.disconnectIndex)
		c.wsConn = nil
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
