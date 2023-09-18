// Copyright 2022 gqzcl <gqzcl@qq.com>. All rights reserved.
// Use of this source code is governed by a MIT style

package websocket

import (
	"fmt"
	"time"

	"github.com/panjf2000/gnet/v2"
)

type ServerOption func(o *Server)

func Address(network string, port int64) ServerOption {
	return func(s *Server) {
		s.address = fmt.Sprintf("%s://:%d", network, port)
	}
}

func WithLoadBalancing(lb gnet.LoadBalancing) ServerOption {
	return func(s *Server) {
		s.lb = lb
	}
}

// ReadTimeout 连接的读超时时间
func ReadTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.readTimeout = timeout
	}
}

// IdleTimeout 连接的空闲超时时间
func IdleTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.idleTimeout = timeout
	}
}

// StopTimeout GNet的停止超时时间
func StopTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.stopTimeout = timeout
	}
}

// TickTime 定时执行时间间隔
func TickTime(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.tickTime = timeout
	}
}

func OnConnectHandle(h OnConnectHandler) ServerOption {
	return func(s *Server) {
		s.onConnectHandler = h
	}
}

func OnMessageHandle(h OnMessageHandler) ServerOption {
	return func(s *Server) {
		s.onMessageHandler = h
	}
}

func OnCloseHandle(h OnCloseHandler) ServerOption {
	return func(s *Server) {
		s.onCloseHandler = h
	}
}

func OnTickHandle(h OnTickHandler) ServerOption {
	return func(s *Server) {
		s.onTickHandler = h
	}
}
