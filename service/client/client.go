package client

import (
	"context"
	"github.com/obnahsgnaw/socketutil/client"
	"github.com/obnahsgnaw/socketutil/codec"
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
	c              *client.Client
	handlers       sync.Map
	cdc            codec.Codec
	pgb            codec.PkgBuilder
	dbd            codec.DataBuilder
	pkgInterceptor PkgInterceptor
	actWatcher     func(action codec.Action, msg string)
	pkgWatcher     func(mtp client.MsgType, msg string, pkg []byte)
	logWatcher     func(level zapcore.Level, msg string)
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
		actWatcher: func(action codec.Action, msg string) {
			log.Println("action[", action.Name, "]", msg)
		},
		pkgWatcher: func(mtp client.MsgType, msg string, pkg []byte) {
			log.Println(mtp.String(), len(pkg), "types pkg:", pkg)
		},
		logWatcher: func(level zapcore.Level, msg string) {
			log.Println(msg)
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
	return nil
}

func (c *Client) Pack(action codec.Action, data codec.DataPtr) ([]byte, error) {
	// data封包
	b, err := c.dbd.Pack(data)
	if err != nil {
		return nil, NewWrappedError("send action["+action.Name+"] failed,pack data failed", err)
	}
	// action封包
	b1, err := c.pgb.Pack(&codec.PKG{
		Action: action.Id,
		Data:   b,
	})
	if err != nil {
		return nil, NewWrappedError("send action["+action.Name+"] failed,pack gateway package failed", err)
	}
	// 拦截器封包
	if c.pkgInterceptor != nil {
		b1, err = c.pkgInterceptor.Encode(b1)
		if err != nil {
			return nil, NewWrappedError("send action["+action.Name+"] failed, interceptor encode package failed", err)
		}
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
		c.logWatcher(zapcore.ErrorLevel, "package dispatcher: dispatch failed, err="+err+", stack="+stack)
	})
	// 沾包拼包
	var err error
	tmp := c.c.Tmp
	c.c.Tmp = nil
	if len(tmp) > 0 {
		pkg = append(tmp, pkg...)
	}
	// 沾包拆包
	c.c.Tmp, err = c.cdc.Unmarshal(pkg, func(codePkg []byte) {
		c.pkgWatcher(client.Receive, "codec package", codePkg)
		// 拦截器解码
		if c.pkgInterceptor != nil {
			var err1 error
			codePkg, err1 = c.pkgInterceptor.Decode(codePkg)
			if err1 != nil {
				c.logWatcher(zapcore.ErrorLevel, "package dispatcher: interceptor decode package failed, err="+err.Error())
				return
			}
		}
		// 网关层的包拆包
		gatewayPackage, err1 := c.pgb.Unpack(codePkg)
		if err1 != nil {
			c.logWatcher(zapcore.ErrorLevel, "package dispatcher: unpack gateway package failed, err="+err.Error())
			return
		}
		// 获取action
		ds, action, handler, ok := c.getHandler(gatewayPackage.Action)
		if !ok {
			c.logWatcher(zapcore.WarnLevel, "package dispatcher: no action["+strconv.Itoa(int(gatewayPackage.Action))+"] handler")
			return
		}
		c.actWatcher(action, "handle start")
		// data 解码
		d := ds()
		if err = c.dbd.Unpack(gatewayPackage.Data, d); err != nil {
			c.actWatcher(action, "data decode failed, err="+err.Error())
			return
		}
		// 处理
		respAction, respData := handler(d)
		if respAction.Id <= 0 {
			c.actWatcher(action, "handle success, but no response")
			return
		}
		// 回复
		if err = c.Send(respAction, respData); err != nil {
			c.logWatcher(zapcore.ErrorLevel, "dispatcher: response action["+action.Name+"] failed,err="+err.Error())

			c.actWatcher(action, "handle success, but response failed, err="+err.Error())
		}

		c.actWatcher(action, "handle success, response action="+respAction.String())
	})
	if err != nil {
		c.logWatcher(zapcore.ErrorLevel, "dispatcher: codec package failed, err="+err.Error())
	}
}
