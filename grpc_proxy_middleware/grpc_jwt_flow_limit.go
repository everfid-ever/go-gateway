package grpc_proxy_middleware

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"go-gateway/dao"
	"go-gateway/public"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"log"
	"strings"
)

// GrpcJwtFlowLimitMiddleware 基于 JWT 的租户级实时 QPS 限流中间件
// 功能：
// 1. 从 Metadata 中读取已鉴权的 App 信息
// 2. 按 App + ClientIP 维度进行实时 QPS 限流
// 3. 防止单个租户的单个客户端瞬时打爆服务
func GrpcJwtFlowLimitMiddleware(serviceDetail *dao.ServiceDetail) func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {

		// ===================== ① 从 Context 中获取 Metadata =====================
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return errors.New("miss metadata from context")
		}

		// ===================== ② 读取上游 JWT 鉴权中写入的 App 信息 =====================
		appInfos := md.Get("app")

		// 若当前请求未携带 App 信息（说明是匿名请求或未开启鉴权）
		// 不做租户级 QPS 限流，直接放行
		if len(appInfos) == 0 {
			if err := handler(srv, ss); err != nil {
				log.Printf("RPC failed with error %v\n", err)
				return err
			}
			return nil
		}

		// ===================== ③ 反序列化 App 信息 =====================
		appInfo := &dao.App{}
		if err := json.Unmarshal([]byte(appInfos[0]), appInfo); err != nil {
			return err
		}

		// ===================== ④ 获取客户端 IP 地址 =====================
		peerCtx, ok := peer.FromContext(ss.Context())
		if !ok {
			return errors.New("peer not found with context")
		}
		peerAddr := peerCtx.Addr.String()
		addrPos := strings.LastIndex(peerAddr, ":")
		clientIP := peerAddr[0:addrPos]

		// ===================== ⑤ 按租户 + IP 做实时 QPS 限流 =====================
		// 例如：每个 App 对单个 IP 限制每秒最大请求数
		if appInfo.Qps > 0 {

			// 限流 Key = appID + clientIP
			clientLimiter, err := public.FlowLimiterHandler.GetLimiter(
				public.FlowAppPrefix+appInfo.AppID+"_"+clientIP,
				float64(appInfo.Qps),
			)
			if err != nil {
				return err
			}

			// 判断当前请求是否超过 QPS 限制
			if !clientLimiter.Allow() {
				return errors.New(
					fmt.Sprintf("%v flow limit %v", clientIP, appInfo.Qps),
				)
			}
		}

		// ===================== ⑥ 放行业务 RPC 处理 =====================
		if err := handler(srv, ss); err != nil {
			log.Printf("RPC failed with error %v\n", err)
			return err
		}

		return nil
	}
}
