package http_proxy_middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go-gateway/dao"
	"go-gateway/middleware"
	"go-gateway/public"
)

// HTTPFlowLimitMiddleware 服务级与客户端 IP 级限流中间件
// 功能：
// 1. 对单个服务进行 QPS 限流
// 2. 对单个客户端 IP + 服务 维度进行 QPS 限流
func HTTPFlowLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 从上下文中获取当前请求对应的服务信息
		serverInterface, ok := c.Get("service")
		if !ok {
			// 未获取到服务信息，直接返回错误并中断请求
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}

		// 类型断言，获取服务详情
		serviceDetail := serverInterface.(*dao.ServiceDetail)

		// =====================================================
		// 1️⃣ 服务级限流（按 ServiceName 维度）
		// =====================================================
		if serviceDetail.AccessControl.ServiceFlowLimit != 0 {

			// 获取该服务对应的限流器
			serviceLimiter, err := public.FlowLimiterHandler.GetLimiter(
				public.FlowServicePrefix+serviceDetail.Info.ServiceName,
				float64(serviceDetail.AccessControl.ServiceFlowLimit),
			)
			if err != nil {
				// 获取限流器失败，返回错误并中断请求
				middleware.ResponseError(c, 5001, err)
				c.Abort()
				return
			}

			// 判断当前请求是否被限流
			if !serviceLimiter.Allow() {
				middleware.ResponseError(
					c,
					5002,
					errors.New(fmt.Sprintf(
						"service flow limit %v",
						serviceDetail.AccessControl.ServiceFlowLimit,
					)),
				)
				c.Abort()
				return
			}
		}

		// =====================================================
		// 2️⃣ 客户端 IP 级限流（按 Service + ClientIP 维度）
		// =====================================================
		if serviceDetail.AccessControl.ClientIPFlowLimit > 0 {

			// key = 服务名 + 客户端 IP
			clientLimiter, err := public.FlowLimiterHandler.GetLimiter(
				public.FlowServicePrefix+
					serviceDetail.Info.ServiceName+
					"_"+c.ClientIP(),
				float64(serviceDetail.AccessControl.ClientIPFlowLimit),
			)
			if err != nil {
				// 获取客户端限流器失败
				middleware.ResponseError(c, 5003, err)
				c.Abort()
				return
			}

			// 判断当前客户端是否被限流
			if !clientLimiter.Allow() {
				middleware.ResponseError(
					c,
					5002,
					errors.New(fmt.Sprintf(
						"%v flow limit %v",
						c.ClientIP(),
						serviceDetail.AccessControl.ClientIPFlowLimit,
					)),
				)
				c.Abort()
				return
			}
		}

		// 未触发任何限流规则，放行请求
		c.Next()
	}
}
