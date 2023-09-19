// Copyright 2022 gqzcl <gqzcl@qq.com>. All rights reserved.
// Use of this source code is governed by a MIT style

package gnetws

import (
	"fmt"
	"time"

	"github.com/panjf2000/gnet/v2"
)

type ServerOption func(o *WsServer)

func Address(network, port string) ServerOption {
	return func(s *WsServer) {
		s.address = fmt.Sprintf("%s://%s", network, port)
	}
}

func WithMaxConn(maxConn int) ServerOption {
	return func(s *WsServer) {
		s.maxConn = maxConn
	}
}

func WithLoadBalancing(lb gnet.LoadBalancing) ServerOption {
	return func(s *WsServer) {
		s.lb = lb
	}
}

// ReadTimeout 连接的读超时时间
func ReadTimeout(timeout time.Duration) ServerOption {
	return func(s *WsServer) {
		s.readTimeout = timeout
	}
}

// IdleTimeout 连接的空闲超时时间
func IdleTimeout(timeout time.Duration) ServerOption {
	return func(s *WsServer) {
		s.idleTimeout = timeout
	}
}

// StopTimeout GNet的停止超时时间
func StopTimeout(timeout time.Duration) ServerOption {
	return func(s *WsServer) {
		s.stopTimeout = timeout
	}
}

// TickTime 定时执行时间间隔
func TickTime(timeout time.Duration) ServerOption {
	return func(s *WsServer) {
		s.tickTime = timeout
	}
}

func OnConnectHandle(h OnConnectHandler) ServerOption {
	return func(s *WsServer) {
		s.onConnectHandler = h
	}
}

func OnMessageHandle(h OnMessageHandler) ServerOption {
	return func(s *WsServer) {
		s.onMessageHandler = h
	}
}

func OnCloseHandle(h OnCloseHandler) ServerOption {
	return func(s *WsServer) {
		s.onCloseHandler = h
	}
}

func OnTickHandle(h OnTickHandler) ServerOption {
	return func(s *WsServer) {
		s.onTickHandler = h
	}
}
