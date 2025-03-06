package client

import (
	"context"
	"log"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	ctx := context.Background()
	cc := New(ctx, "ws", "127.0.0.1:28088/wss",
		Connect(func(index int) {
			log.Println("connect index:", index)
		}),
		Disconnect(func(index int) {
			log.Println("disconnect index:", index)
		}),
		Package(func(mtp MsgType, msg string, pkg []byte) {
			log.Println(mtp.String(), msg, string(pkg))
		}),
		Message(func(pkg []byte) {
			log.Println(string(pkg))
		}),
	)
	cc.Start()

	go func() {
		for {
			cc.Send([]byte("hello world"))
			time.Sleep(time.Second * 5)
		}
	}()
	select {}
}
