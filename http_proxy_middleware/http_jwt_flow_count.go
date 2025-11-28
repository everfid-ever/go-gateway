package http_proxy_middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go-gateway/dao"
	"go-gateway/middleware"
	"go-gateway/public"
)

// HTTPJwtFlowCountMiddleware 租户（JWT 应用）流量统计与日限流中间件
// 功能：
// 1. 按 AppID 维度统计每个租户的请求流量
// 2. 当配置了租户每日请求上限（Qpd）时，进行日级限流控制
func HTTPJwtFlowCountMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 从上下文中获取当前请求对应的应用信息（通常由 JWT 解析中间件注入）
		appInterface, ok := c.Get("app")
		if !ok {
			// 未绑定应用信息，说明该请求不走租户限流，直接放行
			c.Next()
			return
		}

		// 类型断言，获取应用信息
		appInfo := appInterface.(*dao.App)

		// -----------------------------
		// 1️⃣ 获取租户级流量计数器（按 AppID 区分）
		// -----------------------------
		appCounter, err := public.FlowCounterHandler.GetCounter(
			public.FlowAppPrefix + appInfo.AppID,
		)
		if err != nil {
			// 获取租户计数器失败，返回错误并中断请求
			middleware.ResponseError(c, 2002, err)
			c.Abort()
			return
		}

		// 租户请求数 +1
		appCounter.Increase()

		// -----------------------------
		// 2️⃣ 租户日请求量限流（QPD：Query Per Day）
		// -----------------------------
		if appInfo.Qpd > 0 && appCounter.TotalCount > appInfo.Qpd {

			// 超出每日请求上限，触发限流
			middleware.ResponseError(
				c,
				2003,
				errors.New(fmt.Sprintf(
					"租户日请求量限流 limit:%v current:%v",
					appInfo.Qpd,
					appCounter.TotalCount,
				)),
			)
			c.Abort()
			return
		}

		// 未触发限流，放行请求
		c.Next()
	}
}
