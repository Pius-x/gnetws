// Copyright 2022 gqzcl <gqzcl@qq.com>. All rights reserved.
// Use of this source code is governed by a MIT style

package gnetws

import (
	"context"
	"fmt"
	"time"

	"github.com/Pius-x/gnetws/utils"
	"github.com/Pius-x/gnetws/zaplog"
	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	antsPool "github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type WsServer struct {
	gnet.BuiltinEventEngine
	eng        gnet.Engine
	lb         gnet.LoadBalancing
	workerPool *antsPool.Pool
	logger     *zap.Logger

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
	srv.logger, _ = zap.NewProduction()

	for _, opt := range opts {
		opt(srv)
	}

	return srv
}

func (wss *WsServer) Start() {
	err := gnet.Run(wss, wss.address,
		gnet.WithMulticore(true),
		gnet.WithReusePort(true),
		gnet.WithTicker(wss.tickTime > 0),
		gnet.WithLoadBalancing(wss.lb),
	)

	if err != nil {
		panic(errors.Wrap(err, "GNet Start Error"))
	}
}

func (wss *WsServer) Stop(ctx context.Context) {

	// 设置服务器超时时间
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, wss.stopTimeout)

	defer func() {
		wss.logger.Warn("[websocket] server stopping")
		cancel()
	}()

	_ = wss.eng.Stop(ctx)
}

func (wss *WsServer) OnBoot(eng gnet.Engine) gnet.Action {
	wss.eng = eng
	wss.logger.Info(fmt.Sprintf("ws server listening on %s", wss.address))
	return gnet.None
}

func (wss *WsServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	err := utils.Try(func() error {
		if wss.eng.CountConnections() > wss.maxConn {
			return errors.New(fmt.Sprintf("out of maximum, cur conn is: %d", wss.eng.CountConnections()))
		}
		c.SetContext(&WsCodec{connTime: time.Now(), Uuid: uuid.NewString()})
		return nil
	})

	if err != nil {
		wss.logger.Error("OnOpen Err", zaplog.AppendErr(err)...)
		return []byte(err.Error()), gnet.Close
	}

	return nil, gnet.None
}

func (wss *WsServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	err = utils.Try(func() error {
		ws := c.Context().(*WsCodec)

		wss.logger.Info(fmt.Sprintf("conn[%v] disconnected", c.RemoteAddr().String()),
			zap.String(zaplog.TraceId, ws.Uuid),
		)

		if err != nil {
			wss.logger.Warn(fmt.Sprintf("conn[%v] disconnect err", c.RemoteAddr().String()),
				zaplog.AppendErr(err, zap.String(zaplog.TraceId, ws.Uuid))...,
			)
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
			wss.logger.Info(fmt.Sprintf("conn[%v] connected", c.RemoteAddr().String()),
				zap.Int64(zaplog.Latency, time.Since(ws.connTime).Milliseconds()),
				zap.String(zaplog.TraceId, ws.Uuid),
			)
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
			if wss.onMessageHandler != nil {
				return wss.workerPool.Submit(func() {
					if err = wss.onMessageHandler(c, message); err != nil {
						wss.logger.Error("onMessageHandler Err",
							zaplog.AppendErr(err, zap.String(zaplog.TraceId, ws.Uuid))...,
						)
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
		wss.logger.Info(fmt.Sprintf("[connected-count= %v]", wss.eng.CountConnections()))
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
