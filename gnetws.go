// Copyright 2022 gqzcl <gqzcl@qq.com>. All rights reserved.
// Use of this source code is governed by a MIT style

package gnetws

import (
	"context"
	"fmt"
	"time"

	"github.com/Pius-x/gnetws/utils"
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
	err := utils.Try(func() error {
		if wss.eng.CountConnections() > wss.maxConn {
			logging.Warnf("Conn num out of maximum, Current conn num : %d", wss.eng.CountConnections())
			return errors.New("out of maximum")
		}
		c.SetContext(new(WsCodec))
		return nil
	})

	if err != nil {
		return []byte("out of maximum"), gnet.Close
	}

	return nil, gnet.None
}

func (wss *WsServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	err = utils.Try(func() error {
		logging.Infof("conn[%v] disconnected", c.RemoteAddr().String())

		if err != nil {
			logging.Warnf("error occurred on connection=%s, %v\n", c.RemoteAddr().String(), err)
			return err
		}

		if wss.onCloseHandler != nil {
			wss.onCloseHandler(c)
		}

		return nil
	})

	if err != nil {
		return gnet.Close
	}
	return gnet.None
}

func (wss *WsServer) OnTraffic(c gnet.Conn) (action gnet.Action) {

	err := utils.Try(func() error {
		ws := c.Context().(*WsCodec)
		if ws.readBufferBytes(c) == gnet.Close {
			return errors.New("readBufferBytes closed")
		}

		if !ws.upgraded {
			var ok bool
			ok, action = ws.upgrade(c)
			if !ok {
				return nil
			}
			if wss.onConnectHandler != nil {
				wss.onConnectHandler(c, ws)
			}
		}

		if ws.buf.Len() <= 0 {
			return nil
		}
		messages, err := ws.Decode(c)
		if err != nil {
			return err
		}
		if messages == nil {
			return nil
		}
		for _, message := range messages {
			//msgLen := len(message.Payload)
			//if msgLen > 128 {
			//	logging.Infof("conn[%v] receive [op=%v] [msg=%v..., len=%d]", c.RemoteAddr().String(), message.OpCode, string(message.Payload[:128]), len(message.Payload))
			//} else {
			//	logging.Infof("conn[%v] receive [op=%v] [msg=%v, len=%d]", c.RemoteAddr().String(), message.OpCode, string(message.Payload), len(message.Payload))
			//}

			if wss.onMessageHandler != nil {
				return wss.workerPool.Submit(func() {
					if err = wss.onMessageHandler(c, message); err != nil {
						logging.Error(err)
					}
				})
			}
		}

		return nil
	})

	if err != nil {
		return gnet.Close
	}
	return gnet.None
}

func (wss *WsServer) OnTick() (delay time.Duration, action gnet.Action) {
	err := utils.Try(func() error {
		logging.Infof("[connected-count=%v]", wss.eng.CountConnections())
		if wss.onTickHandler != nil {
			wss.onTickHandler()
		}

		return nil
	})

	if err != nil {
		return wss.tickTime, gnet.Close
	}
	return wss.tickTime, gnet.None
}
