package client

import (
	"context"
	"github.com/obnahsgnaw/socketutil/client"
	"github.com/obnahsgnaw/socketutil/codec"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"strconv"
	"sync"
	"time"
)

// DataStructure 提供data的数据结构
type DataStructure func() codec.DataPtr

// Handler action的处理器， rqData即DataStructure提供的数据结构解析后的结构
type Handler func(rqData codec.DataPtr) (respAction codec.Action, respData codec.DataPtr)

type Client struct {
	c        *client.Client
	handlers sync.Map
	cdc      codec.Codec
	pgb      codec.PkgBuilder
	dbd      codec.DataBuilder
	watcher  func(level zapcore.Level, msg string, data ...zap.Field)
}

type listenHandler struct {
	action    codec.Action
	structure DataStructure
	handler   Handler
}

func New(ctx context.Context, network string, host string, cdc codec.Codec, pgb codec.PkgBuilder, dbd codec.DataBuilder, options ...Option) *Client {
	c := &Client{
		c:   client.New(ctx, network, host),
		cdc: cdc,
		pgb: pgb,
		dbd: dbd,
		watcher: func(level zapcore.Level, msg string, data ...zap.Field) {
			log.Println(level.String(), msg)
		},
	}
	c.With(options...)
	c.c.With(client.Message(c.dispatch))

	return c
}

func (c *Client) With(options ...Option) {
	for _, o := range options {
		o(c)
	}
}

func (c *Client) Listen(action codec.Action, structure DataStructure, handler Handler) {
	c.addHandler(listenHandler{
		action:    action,
		structure: structure,
		handler:   handler,
	})
}

func (c *Client) Send(action codec.Action, data codec.DataPtr) error {
	b2, err := c.Pack(action, data)
	if err != nil {
		return err
	}
	err = c.c.Send(b2)
	if err != nil {
		return NewWrappedError("send action["+action.Name+"] failed,send failed", err)
	}
	c.watcher(zapcore.DebugLevel, "send action["+action.Name+"] success", zap.ByteString("pkg", b2))
	return nil
}

func (c *Client) Pack(action codec.Action, data codec.DataPtr) ([]byte, error) {
	// data封包
	b, err := c.dbd.Pack(data)
	if err != nil {
		return nil, NewWrappedError("send action["+action.Name+"] failed,pack data failed", err)
	}
	// todo encrypt
	// action封包
	b1, err := c.pgb.Pack(&codec.PKG{
		Action: action.Id,
		Data:   b,
	})
	if err != nil {
		return nil, NewWrappedError("send action["+action.Name+"] failed,pack gateway package failed", err)
	}
	// codec封包
	b2, err := c.cdc.Marshal(b1)
	if err != nil {
		return nil, NewWrappedError("send action["+action.Name+"] failed,pack codec package failed", err)
	}

	return b2, nil
}

func (c *Client) Heartbeat(pkg []byte, interval time.Duration) {
	c.c.Heartbeat(pkg, interval)
}

func (c *Client) Start() {
	c.c.Start()
}

func (c *Client) Stop() {
	c.c.Stop()
}

func (c *Client) addHandler(handler listenHandler) {
	c.handlers.Store(handler.action.Id, handler)
}

func (c *Client) delHandler(id codec.ActionId) {
	c.handlers.Delete(id)
}

func (c *Client) getHandler(id codec.ActionId) (DataStructure, codec.Action, Handler, bool) {
	h, ok := c.handlers.Load(id)
	if ok {
		h1 := h.(listenHandler)
		return h1.structure, h1.action, h1.handler, true
	}

	return nil, codec.Action{}, nil, false
}

func (c *Client) dispatch(pkg []byte) {
	defer RecoverHandler("client server dispatcher", func(err, stack string) {
		c.watcher(zapcore.ErrorLevel, "dispatch failed, err="+err+", stack="+stack)
	})
	c.watcher(zapcore.DebugLevel, "dispatcher: received raw package", zap.ByteString("pkg", pkg))
	// 沾包拼包
	var err error
	tmp := c.c.Tmp
	c.c.Tmp = nil
	if len(tmp) > 0 {
		pkg = append(tmp, pkg...)
		c.watcher(zapcore.DebugLevel, "dispatcher: withed tmp package", zap.ByteString("pkg", pkg))
	}
	// 沾包拆包
	c.c.Tmp, err = c.cdc.Unmarshal(pkg, func(codePkg []byte) {
		c.watcher(zapcore.DebugLevel, "dispatcher: received package", zap.ByteString("pkg", codePkg))
		// 网关层的包拆包
		gatewayPackage, err1 := c.pgb.Unpack(codePkg)
		if err1 != nil {
			c.watcher(zapcore.ErrorLevel, "dispatcher: unpack gateway package failed, err="+err.Error())
			return
		}
		c.watcher(zapcore.DebugLevel, "dispatcher: received action: "+strconv.Itoa(int(gatewayPackage.Action)))
		// todo decrypt
		// 获取action
		ds, action, handler, ok := c.getHandler(codec.ActionId(gatewayPackage.Action))
		if !ok {
			c.watcher(zapcore.WarnLevel, "dispatcher: no action["+strconv.Itoa(int(gatewayPackage.Action))+"] handler")
			return
		}
		c.watcher(zapcore.InfoLevel, "dispatcher: handle action="+action.String())
		// data 解码
		d := ds()
		if err = c.dbd.Unpack(gatewayPackage.Data, d); err != nil {
			c.watcher(zapcore.ErrorLevel, "dispatcher: action data decode failed, err="+err.Error())
			return
		}
		// 处理
		respAction, respData := handler(d)
		if respAction.Id <= 0 {
			c.watcher(zapcore.InfoLevel, "dispatcher: handle success, no response")
			return
		}
		c.watcher(zapcore.InfoLevel, "dispatcher: handle success, response action="+respAction.String())
		// 回复
		if err = c.Send(respAction, respData); err != nil {
			c.watcher(zapcore.ErrorLevel, "dispatcher: "+err.Error())
		}
	})
	if err != nil {
		c.watcher(zapcore.ErrorLevel, "dispatcher: unpack codec package failed, err="+err.Error())
	}
}

// todo gateway err, auth, crypt
