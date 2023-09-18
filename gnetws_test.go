package gnetws

import (
	"fmt"
	"testing"

	"github.com/panjf2000/gnet/v2"
)

func TestRun(t *testing.T) {

	Run(
		Address("tcp", 7700),
		OnConnectHandle(ConnectHandler),
		OnMessageHandle(MessageHandler),
		OnCloseHandle(CloseHandler),
		OnTickHandle(TickHandler),
	)
}

func ConnectHandler(c gnet.Conn, ctx *WsContext) {

	fmt.Println("ConnectHandler", c, ctx)

}

func MessageHandler(c gnet.Conn, message []byte) error {
	fmt.Println("MessageHandler", c, message)
	return nil
}

func CloseHandler(c gnet.Conn) {
	fmt.Println("CloseHandler", c)
}

func TickHandler() {
	fmt.Println("TickHandler")
}
