// Copyright 2022 gqzcl <gqzcl@qq.com>. All rights reserved.
// Use of this source code is governed by a MIT style

package gnetws

import (
	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet/v2"
)

// OnConnectHandler 建立连接时处理
type OnConnectHandler func(gnet.Conn, *WsCodec)

// OnCloseHandler 断开连接时处理
type OnCloseHandler func(gnet.Conn)

// OnMessageHandler 收到消息时处理
type OnMessageHandler func(gnet.Conn, wsutil.Message) error

// OnErrorHandler 出现错误时处理
type OnErrorHandler func(gnet.Conn, error)

// OnTickHandler 定时任务
type OnTickHandler func()
