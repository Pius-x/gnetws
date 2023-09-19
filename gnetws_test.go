package gnetws

import (
	"fmt"
	"testing"

	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet/v2"
)

func TestRun(t *testing.T) {

	svr := Run(
		Address("tcp", ":7788"),
		OnConnectHandle(ConnectHandler),
		OnMessageHandle(MessageHandler),
		OnCloseHandle(CloseHandler),
		OnTickHandle(TickHandler),
		WithMaxConn(100000),
	)

	svr.Start()
}

func ConnectHandler(c gnet.Conn, ctx *WsCodec) {

	fmt.Println("ConnectHandler", c, ctx)

}

func MessageHandler(c gnet.Conn, message wsutil.Message) error {
	fmt.Println("MessageHandler", c, message)
	return nil
}

func CloseHandler(c gnet.Conn) {
	fmt.Println("CloseHandler", c)
}

func TickHandler() {
	fmt.Println("TickHandler")
}
