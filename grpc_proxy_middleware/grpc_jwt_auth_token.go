package grpc_proxy_middleware

import (
	"github.com/pkg/errors"
	"go-gateway/dao"
	"go-gateway/public"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"log"
	"strings"
)

// GrpcJwtAuthTokenMiddleware gRPC JWT 鉴权中间件
// 功能：
// 1. 从 gRPC Metadata 中提取 Authorization Token
// 2. 解析 JWT，校验合法性
// 3. 根据 Token 中的 Issuer 匹配合法 App
// 4. 将 App 信息写入 Metadata 供后续服务使用
// 5. 未通过鉴权时直接拦截请求
func GrpcJwtAuthTokenMiddleware(serviceDetail *dao.ServiceDetail) func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// 返回一个标准的 gRPC Stream 拦截器函数
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {

		// ===================== ① 从 Context 中提取 gRPC Metadata =====================
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return errors.New("miss metadata from context")
		}

		// ===================== ② 读取 Authorization 头 =====================
		authToken := ""
		auths := md.Get("authorization")
		if len(auths) > 0 {
			authToken = auths[0]
		}

		// 去除 "Bearer " 前缀，提取纯 Token
		token := strings.ReplaceAll(authToken, "Bearer ", "")

		// 标记是否匹配到合法 App
		appMatched := false

		// ===================== ③ 解析并验证 JWT Token =====================
		if token != "" {
			// 解析 JWT，获取 Claims
			claims, err := public.JwtDecode(token)
			if err != nil {
				return errors.WithMessage(err, "JwtDecode")
			}

			// 获取系统中所有已注册的 App 列表
			appList := dao.AppManagerHandler.GetAppList()

			// 遍历所有 App，匹配 Issuer（通常代表 AppID）
			for _, appInfo := range appList {
				if appInfo.AppID == claims.Issuer {

					// 将匹配到的 App 信息写入 Metadata
					// 供后续 RPC 业务逻辑使用
					md.Set("app", public.Obj2Json(appInfo))

					appMatched = true
					break
				}
			}
		}

		// ===================== ④ 是否开启鉴权校验 =====================
		// 如果服务开启了鉴权，并且 Token 未匹配到合法 App，则拒绝请求
		if serviceDetail.AccessControl.OpenAuth == 1 && !appMatched {
			return errors.New("not match valid app")
		}

		// ===================== ⑤ 放行执行业务 RPC =====================
		if err := handler(srv, ss); err != nil {
			log.Printf("GrpcJwtAuthTokenMiddleware failed with error %v\n", err)
			return err
		}

		return nil
	}
}
