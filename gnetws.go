// Copyright 2022 gqzcl <gqzcl@qq.com>. All rights reserved.
// Use of this source code is governed by a MIT style

package gnetws

import (
	"context"
	"fmt"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
	antsPool "github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

type WsServer struct {
	gnet.BuiltinEventEngine
	eng        gnet.Engine
	lb         gnet.LoadBalancing
	workerPool *antsPool.Pool

	address     string
	maxConn     int
	tickTime    time.Duration
	readTimeout time.Duration
	idleTimeout time.Duration
	stopTimeout time.Duration

	onConnectHandler OnConnectHandler
	onMessageHandler OnMessageHandler
	onCloseHandler   OnCloseHandler
	onTickHandler    OnTickHandler
}

func Run(opts ...ServerOption) *WsServer {
	srv := &WsServer{
		maxConn:     1000,
		readTimeout: 2 * time.Second,
		idleTimeout: 30 * time.Second,
		stopTimeout: 10 * time.Second,
		tickTime:    10 * time.Second,

		lb:         gnet.LeastConnections,
		workerPool: antsPool.Default(),
	}

	for _, opt := range opts {
		opt(srv)
	}

	return srv
}

func (wss *WsServer) Start() {
	err := gnet.Run(wss, wss.address,
		gnet.WithMulticore(true),
		gnet.WithReusePort(true),
		gnet.WithTicker(lo.Ternary(wss.tickTime > 0, true, false)),
		gnet.WithLoadBalancing(wss.lb),
	)

	if err != nil {
		panic(errors.Wrap(err, "GNet Start Error"))
	}
}

func (wss *WsServer) Stop() {

	// 设置服务器超时时间
	ctx, cancel := context.WithTimeout(context.Background(), wss.stopTimeout)
	defer func() {
		logging.Infof("[websocket] server stopping")
		cancel()
	}()
	fmt.Println("Ws svr Stop....")
	_ = wss.eng.Stop(ctx)
}

func (wss *WsServer) OnBoot(eng gnet.Engine) gnet.Action {
	wss.eng = eng
	logging.Infof("ws server listening on %s", wss.address)
	return gnet.None
}

func (wss *WsServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	if wss.eng.CountConnections() > wss.maxConn {
		logging.Warnf("Conn num out of maximum, Current conn num : %d", wss.eng.CountConnections())
		return []byte("out of maximum"), gnet.Close
	}
	c.SetContext(new(WsCodec))
	return nil, gnet.None
}

func (wss *WsServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	if err != nil {
		logging.Warnf("error occurred on connection=%wss, %v\n", c.RemoteAddr().String(), err)
	}

	if wss.onCloseHandler != nil {
		wss.onCloseHandler(c)
	}
	logging.Infof("conn[%v] disconnected", c.RemoteAddr().String())
	return gnet.None
}

func (wss *WsServer) OnTraffic(c gnet.Conn) (action gnet.Action) {
	logging.Warnf("Current conn num : %d", wss.eng.CountConnections())

	ws := c.Context().(*WsCodec)
	if ws.readBufferBytes(c) == gnet.Close {
		return gnet.Close
	}

	if !ws.upgraded {
		var ok bool
		ok, action = ws.upgrade(c)
		if !ok {
			return
		}
		if wss.onConnectHandler != nil {
			wss.onConnectHandler(c, ws)
		}
	}

	if ws.buf.Len() <= 0 {
		return gnet.None
	}
	messages, err := ws.Decode(c)
	if err != nil {
		return gnet.Close
	}
	if messages == nil {
		return
	}
	for _, message := range messages {
		msgLen := len(message.Payload)
		if msgLen > 128 {
			logging.Infof("conn[%v] receive [op=%v] [msg=%v..., len=%d]", c.RemoteAddr().String(), message.OpCode, string(message.Payload[:128]), len(message.Payload))
		} else {
			logging.Infof("conn[%v] receive [op=%v] [msg=%v, len=%d]", c.RemoteAddr().String(), message.OpCode, string(message.Payload), len(message.Payload))
		}

		if wss.onMessageHandler != nil {
			err = wss.workerPool.Submit(func() {
				if err = wss.onMessageHandler(c, message); err != nil {
					fmt.Println("onMessageHandler Err", err)
				}
			})
			if err != nil {
				fmt.Println("errerrerrerrerrerrerrerrerrerrerrerrerrerrerrerrerrerr", err)
				return gnet.Close
			}
		}
	}
	return gnet.None
}

func (wss *WsServer) OnTick() (delay time.Duration, action gnet.Action) {
	logging.Infof("[connected-count=%v]", wss.eng.CountConnections())
	if wss.onTickHandler != nil {
		wss.onTickHandler()
	}

	return wss.tickTime, gnet.None
}
