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

// GrpcFlowLimitMiddleware gRPC 流量限流中间件
// 功能：
// 1. 对整个服务做 QPS 限流（ServiceFlowLimit）
// 2. 对单个客户端 IP 做 QPS 限流（ClientIPFlowLimit）
func GrpcFlowLimitMiddleware(serviceDetail *dao.ServiceDetail) func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// 返回一个标准 gRPC StreamServerInterceptor
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {

		// ===================== ① 服务级限流 =====================
		// 如果配置了服务级限流（QPS > 0）
		if serviceDetail.AccessControl.ServiceFlowLimit != 0 {

			// 获取该服务对应的限流器（key = serviceName）
			serviceLimiter, err := public.FlowLimiterHandler.GetLimiter(
				public.FlowServicePrefix+serviceDetail.Info.ServiceName,
				float64(serviceDetail.AccessControl.ServiceFlowLimit),
			)
			if err != nil {
				return err
			}

			// 判断是否允许当前请求通过
			if !serviceLimiter.Allow() {
				// 超过服务级限流阈值，直接拒绝
				return errors.New(
					fmt.Sprintf("service flow limit %v",
						serviceDetail.AccessControl.ServiceFlowLimit),
				)
			}
		}

		// ===================== ② 获取客户端 IP =====================
		peerCtx, ok := peer.FromContext(ss.Context())
		if !ok {
			return errors.New("peer not found with context")
		}

		// peer 地址格式一般为：IP:PORT
		peerAddr := peerCtx.Addr.String()
		addrPos := strings.LastIndex(peerAddr, ":")
		clientIP := peerAddr[0:addrPos] // 只取 IP 部分

		// ===================== ③ 客户端 IP 级限流 =====================
		// 如果配置了按 IP 限流
		if serviceDetail.AccessControl.ClientIPFlowLimit > 0 {

			// 获取客户端 IP 专属的限流器
			clientLimiter, err := public.FlowLimiterHandler.GetLimiter(
				public.FlowServicePrefix+
					serviceDetail.Info.ServiceName+"_"+clientIP,
				float64(serviceDetail.AccessControl.ClientIPFlowLimit),
			)
			if err != nil {
				return err
			}

			// 判断当前 IP 是否超限
			if !clientLimiter.Allow() {
				return errors.New(
					fmt.Sprintf("%v flow limit %v",
						clientIP,
						serviceDetail.AccessControl.ClientIPFlowLimit),
				)
			}
		}

		// ===================== ④ 通过限流，放行执行真正的 RPC 处理 =====================
		if err := handler(srv, ss); err != nil {
			log.Printf("GrpcFlowLimitMiddleware failed with error %v\n", err)
			return err
		}

		return nil
	}
}
