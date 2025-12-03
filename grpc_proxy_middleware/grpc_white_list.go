package grpc_proxy_middleware

import (
	"fmt"
	"github.com/pkg/errors"
	"go-gateway/dao"
	"go-gateway/public"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"log"
	"strings"
)

// GrpcWhiteListMiddleware 创建一个 gRPC 流式调用的“IP 白名单访问控制中间件”
// 作用：在每个 gRPC Stream 请求进入业务处理前，
//
//	校验客户端 IP 是否在服务配置的白名单中，
//	如果不在白名单内则直接拒绝请求
func GrpcWhiteListMiddleware(serviceDetail *dao.ServiceDetail) func(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {

	// 返回真正被 gRPC Server 注册执行的 Stream 拦截函数
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {

		// -----------------------------
		// 1. 解析服务配置中的白名单 IP 列表
		// -----------------------------
		iplist := []string{}
		if serviceDetail.AccessControl.WhiteList != "" {
			// 多个 IP 使用逗号分隔，拆分为切片
			iplist = strings.Split(
				serviceDetail.AccessControl.WhiteList,
				",",
			)
		}

		// -----------------------------
		// 2. 从 gRPC 上下文中获取客户端连接信息（peer）
		// -----------------------------
		peerCtx, ok := peer.FromContext(ss.Context())
		if !ok {
			// 如果未能从上下文中获取到客户端信息，说明连接异常，直接返回错误
			return errors.New("peer not found with context")
		}

		// 获取客户端的远程地址，格式一般为 "IP:端口"
		peerAddr := peerCtx.Addr.String()

		// -----------------------------
		// 3. 从 "IP:Port" 格式中解析真实客户端 IP
		// -----------------------------
		addrPos := strings.LastIndex(peerAddr, ":")
		clientIP := peerAddr[0:addrPos]

		// -----------------------------
		// 4. 白名单校验逻辑
		// 触发条件：
		// - OpenAuth == 1 ：开启访问控制
		// - 白名单非空   ：配置了白名单 IP 列表
		// -----------------------------
		if serviceDetail.AccessControl.OpenAuth == 1 && len(iplist) > 0 {

			// 如果当前客户端 IP 不在白名单中，则拒绝访问
			if !public.InStringSlice(iplist, clientIP) {
				return errors.New(fmt.Sprintf(
					"%s not in white ip list", clientIP,
				))
			}
		}

		// -----------------------------
		// 5. 白名单校验通过，继续执行真正的 gRPC 业务处理逻辑
		// -----------------------------
		if err := handler(srv, ss); err != nil {
			// 业务处理过程中出现错误，记录日志并返回错误
			log.Printf("RPC failed with error %v\n", err)
			return err
		}

		// -----------------------------
		// 6. 正常执行完成，返回成功
		// -----------------------------
		return nil
	}
}
