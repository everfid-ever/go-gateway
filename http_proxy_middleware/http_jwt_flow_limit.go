package http_proxy_middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go-gateway/dao"
	"go-gateway/middleware"
	"go-gateway/public"
)

// HTTPJwtFlowLimitMiddleware 租户级 QPS 限流中间件（基于 JWT 解析出的 App）
// 功能：
// 1. 根据 JWT 解析出的 App 信息进行限流
// 2. 按 “AppID + ClientIP” 维度做 QPS 控制
func HTTPJwtFlowLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 从上下文中获取租户（App）信息，该信息通常由 JWT 解析中间件注入
		appInterface, ok := c.Get("app")
		if !ok {
			// 未登录或未携带 JWT，不做租户限流，直接放行
			c.Next()
			return
		}

		// 类型断言，获取 App 实体信息
		appInfo := appInterface.(*dao.App)

		// =====================================================
		// 1️⃣ 租户级 QPS 限流（AppID + ClientIP 维度）
		// =====================================================
		if appInfo.Qps > 0 {

			// 生成限流 key：AppID + 客户端 IP
			clientLimiter, err := public.FlowLimiterHandler.GetLimiter(
				public.FlowAppPrefix+appInfo.AppID+"_"+c.ClientIP(),
				float64(appInfo.Qps),
			)
			if err != nil {
				// 获取限流器失败
				middleware.ResponseError(c, 5001, err)
				c.Abort()
				return
			}

			// 判断当前请求是否超过租户 QPS 限制
			if !clientLimiter.Allow() {
				middleware.ResponseError(
					c,
					5002,
					errors.New(fmt.Sprintf(
						"%v flow limit %v",
						c.ClientIP(),
						appInfo.Qps,
					)),
				)
				c.Abort()
				return
			}
		}

		// 未触发限流规则，继续后续中间件或业务处理
		c.Next()
	}
}
