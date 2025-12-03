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

// GrpcBlackListMiddleware 创建一个 gRPC 流式服务的“黑白名单 IP 访问控制中间件”
// 作用：在每一次 gRPC Stream 请求进入真正业务处理前，
//
//	根据 serviceDetail 中配置的黑白名单规则，决定是否放行该客户端 IP
func GrpcBlackListMiddleware(serviceDetail *dao.ServiceDetail) func(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {

	// 返回真正被 gRPC Server 注册的 Stream 拦截函数
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {

		// -----------------------------
		// 1. 解析白名单 IP 列表
		// -----------------------------
		whiteIpList := []string{}
		if serviceDetail.AccessControl.WhiteList != "" {
			// 配置中多个 IP 以逗号分隔，拆分成 slice
			whiteIpList = strings.Split(
				serviceDetail.AccessControl.WhiteList,
				",",
			)
		}

		// -----------------------------
		// 2. 从 gRPC 上下文中获取客户端连接信息（peer）
		// -----------------------------
		peerCtx, ok := peer.FromContext(ss.Context())
		if !ok {
			// 如果上下文中拿不到 peer，说明此请求异常，直接拒绝
			return errors.New("peer not found with context")
		}

		// peerCtx.Addr 形如 "192.168.1.10:54321"
		peerAddr := peerCtx.Addr.String()

		// -----------------------------
		// 3. 从 "IP:端口" 中提取出客户端真实 IP
		// -----------------------------
		addrPos := strings.LastIndex(peerAddr, ":")
		clientIP := peerAddr[0:addrPos]

		// -----------------------------
		// 4. 解析黑名单 IP 列表
		// -----------------------------
		blackIpList := []string{}
		if serviceDetail.AccessControl.BlackList != "" {
			blackIpList = strings.Split(
				serviceDetail.AccessControl.BlackList,
				",",
			)
		}

		// -----------------------------
		// 5. 黑名单校验逻辑
		// 触发条件：
		// - OpenAuth == 1      ：开启访问控制
		// - 黑名单非空         ：配置了黑名单
		// - 白名单为空         ：未配置白名单
		// -----------------------------
		if serviceDetail.AccessControl.OpenAuth == 1 &&
			len(blackIpList) > 0 &&
			len(whiteIpList) == 0 {

			// 判断当前客户端 IP 是否在黑名单中
			if public.InStringSlice(blackIpList, clientIP) {
				// 命中黑名单，直接拒绝请求
				return errors.New(fmt.Sprintf(
					"%s in black ip list", clientIP,
				))
			}
		}

		// -----------------------------
		// 6. 黑名单校验通过，继续执行真正的 gRPC 业务处理逻辑
		// -----------------------------
		if err := handler(srv, ss); err != nil {
			// 业务处理失败，记录日志并返回错误
			log.Printf("RPC failed with error %v\n", err)
			return err
		}

		// -----------------------------
		// 7. 正常执行完成
		// -----------------------------
		return nil
	}
}
