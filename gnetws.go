// Copyright 2022 gqzcl <gqzcl@qq.com>. All rights reserved.
// Use of this source code is governed by a MIT style

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Pius-x/gnetws/utils"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
	antsPool "github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

type Server struct {
	gnet.BuiltinEventEngine
	eng        gnet.Engine
	lb         gnet.LoadBalancing
	workerPool *antsPool.Pool

	address     string
	tickTime    time.Duration
	readTimeout time.Duration
	idleTimeout time.Duration
	stopTimeout time.Duration

	onConnectHandler OnConnectHandler
	onMessageHandler OnMessageHandler
	onCloseHandler   OnCloseHandler
	onTickHandler    OnTickHandler
}

type WsContext struct {
	upgraded bool   // 链接是否升级
	uuid     string // 唯一ID
}

func Run(opts ...ServerOption) *Server {
	srv := &Server{
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

	srv.start()

	return srv
}

func (s *Server) start() {
	err := gnet.Run(s, s.address,
		gnet.WithMulticore(true),
		gnet.WithReusePort(true),
		gnet.WithLockOSThread(true),
		gnet.WithTicker(lo.Ternary(s.tickTime > 0, true, false)),
		gnet.WithLoadBalancing(s.lb),
		gnet.WithTCPNoDelay(gnet.TCPDelay),
		gnet.WithTCPKeepAlive(time.Minute),
	)

	if err != nil {
		panic(errors.Wrap(err, "GNet Start Error"))
	}
}

func (s *Server) Stop() error {

	// 设置服务器超时时间
	ctx, cancel := context.WithTimeout(context.Background(), s.stopTimeout)
	defer func() {
		logging.Infof("[websocket] server stopping")
		cancel()
	}()

	return s.eng.Stop(ctx)
}

func (s *Server) OnBoot(eng gnet.Engine) gnet.Action {
	s.eng = eng
	logging.Infof("ws server with multi-core=%t is listening on %s", true, s.address)
	return gnet.None
}

func (s *Server) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	c.SetContext(new(WsContext))
	return nil, gnet.None
}

func (s *Server) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	if err != nil {
		logging.Warnf("error occurred on connection=%s, %v\n", c.RemoteAddr().String(), err)
	}

	if s.onCloseHandler != nil {
		s.onCloseHandler(c)
	}
	logging.Infof("conn[%v] disconnected", c.RemoteAddr().String())
	return gnet.None
}

func (s *Server) OnTraffic(c gnet.Conn) gnet.Action {

	wsCtx := c.Context().(*WsContext)
	if !wsCtx.upgraded {
		if _, err := ws.Upgrade(c); err != nil {
			logging.Infof("conn[%v] [err=%v]", c.RemoteAddr().String(), err.Error())

			_ = c.Close()
			return gnet.Close
		}
		wsCtx.upgraded = true
		wsCtx.uuid = uuid.NewString()

		if s.onConnectHandler != nil {
			s.onConnectHandler(c, wsCtx)
		}

		return gnet.None
	}

	msg, op, err := wsutil.ReadClientData(c)
	if err != nil {
		if _, ok := err.(wsutil.ClosedError); !ok {
			logging.Infof("conn[%v] [err=%v]", c.RemoteAddr().String(), err.Error())
		}
		return gnet.Close
	}
	logging.Infof("conn[%v] receive [op=%v] [msg=%v]", c.RemoteAddr().String(), op, string(msg))

	err = s.workerPool.Submit(func() {
		if s.onMessageHandler != nil {
			err = utils.Try(func() error {
				return s.onMessageHandler(c, msg)
			})
			if err != nil {
				fmt.Println("onMessageHandler Err", err)
			}
		}
	})
	if err != nil {
		return gnet.Close
	}

	return gnet.None
}

func (s *Server) OnTick() (delay time.Duration, action gnet.Action) {
	logging.Infof("[connected-count=%v]", s.eng.CountConnections())
	if s.onTickHandler != nil {
		s.onTickHandler()
	}

	return s.tickTime, gnet.None
}
