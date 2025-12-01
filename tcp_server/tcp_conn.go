package tcp_server

import (
	"context"
	"fmt"
	"net"
	"runtime"
)

// tcpKeepAliveListener 用于对 net.TCPListener 进行包装
// 目的是覆写 Accept 方法，以便在接受到的连接上设置 TCP KeepAlive 选项
type tcpKeepAliveListener struct {
	*net.TCPListener // 嵌入 net.TCPListener，以便继承其方法
}

// Accept 方法覆写 net.TCPListener 的 Accept 方法
// 这里将底层 TCPListener 的 AccepTCP() 暴露出来，可以进行 TCP 参数控制
// todo 思考点：继承方法覆写方法的时候，只要使用非接口指针
func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}

	// 后续可设置 keep-alive，例如：
	// tc.SetKeepAlive(true)
	// tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

// contextKey 用于 context 中的键类型避免 key 冲突
type contextKey struct {
	name string
}

// 自定义 String 方法，便于调试打印时可读性更高
func (k *contextKey) String() string {
	return "tcp_proxy context value " + k.name
}

// conn 表示一个 TCP 连接会话
type conn struct {
	server     *TcpServer         // 所属的 TCP 服务器（用于回调 Handler）
	cancelCtx  context.CancelFunc // 用于关闭连接上下文
	rwc        net.Conn           // 底层 TCP 连接（Reader/Writer/Closer）
	remoteAddr string             // 记录远端地址（日志使用）
}

// close 负责关闭实际连接
func (c *conn) close() {
	c.rwc.Close()
}

// serve 启动连接的处理逻辑
func (c *conn) serve(ctx context.Context) {
	defer func() {
		// 恢复 panic，避免进程挂掉
		if err := recover(); err != nil && err != ErrAbortHandler {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			fmt.Printf("tcp: panic serving %v: %v\n%s", c.remoteAddr, err, buf)
		}
		c.close()
	}()

	c.remoteAddr = c.rwc.RemoteAddr().String() // 客户端 IP/端口

	// 绑定 context，传递本地监听地址信息，实现类似 http.Server 的 requestContext 设计
	ctx = context.WithValue(ctx, LocalAddrContextKey, c.rwc.LocalAddr())

	// 没有 Handler 直接 panic
	if c.server.Handler == nil {
		panic("tcp: no Handler set on TcpServer")
	}

	// 回调用户自定义处理逻辑，类似 http.Server 的 ServeHTTP
	c.server.Handler.ServeTCP(ctx, c.rwc)
}
