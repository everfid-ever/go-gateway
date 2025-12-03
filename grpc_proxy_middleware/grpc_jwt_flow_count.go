package grpc_proxy_middleware

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"go-gateway/dao"
	"go-gateway/public"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"log"
)

// GrpcJwtFlowCountMiddleware 基于 JWT 的租户流量统计 & 日请求量限流中间件
// 功能：
// 1. 从 gRPC Metadata 中获取已鉴权后的 App 信息
// 2. 按 App 维度进行调用次数统计
// 3. 支持按租户(日维度)进行请求量限制（QPD：Queries Per Day）
// 4. 超过限额直接拒绝请求
func GrpcJwtFlowCountMiddleware(serviceDetail *dao.ServiceDetail) func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
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

		// ===================== ② 读取上游 JWT 鉴权中设置的 App 信息 =====================
		appInfos := md.Get("app")

		// 如果当前请求没有携带 App 信息（说明未开启鉴权或匿名访问）
		// 直接放行，不做租户流量统计
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

		// ===================== ④ 获取当前 App 对应的流量计数器 =====================
		appCounter, err := public.FlowCounterHandler.GetCounter(
			public.FlowAppPrefix + appInfo.AppID,
		)
		if err != nil {
			return err
		}

		// 当前 App 调用次数 +1
		appCounter.Increase()

		// ===================== ⑤ 租户日请求量（QPD）限流 =====================
		// 如果配置了 QPD 且已超过当日最大请求量
		if appInfo.Qpd > 0 && appCounter.TotalCount > appInfo.Qpd {
			return errors.New(
				fmt.Sprintf(
					"租户日请求量限流 limit:%v current:%v",
					appInfo.Qpd,
					appCounter.TotalCount,
				),
			)
		}

		// ===================== ⑥ 放行业务 RPC 处理 =====================
		if err := handler(srv, ss); err != nil {
			log.Printf("RPC failed with error %v\n", err)
			return err
		}

		return nil
	}
}
