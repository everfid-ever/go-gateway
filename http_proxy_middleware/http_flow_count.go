package http_proxy_middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go-gateway/dao"
	"go-gateway/middleware"
	"go-gateway/public"
)

// HTTPFlowCountMiddleware 流量统计中间件
// 功能：对请求流量进行统计，支持全站级与服务级两种维度
// 统计维度说明：
// 1. 全站流量统计（public.FlowTotal）
// 2. 单个服务维度统计（按 ServiceName 区分）
func HTTPFlowCountMiddleware() gin.HandlerFunc {
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

		// -----------------------------
		// 1️⃣ 全站流量统计
		// -----------------------------
		// 获取全站流量计数器
		totalCounter, err := public.FlowCounterHandler.GetCounter(public.FlowTotal)
		if err != nil {
			// 获取计数器失败，返回错误并终止请求
			middleware.ResponseError(c, 4001, err)
			c.Abort()
			return
		}

		// 全站请求数 +1
		totalCounter.Increase()

		// 可选：按天统计（调试用）
		// dayCount, _ := totalCounter.GetDayData(time.Now())
		// fmt.Printf("totalCounter qps:%v, dayCount:%v", totalCounter.QPS, dayCount)

		// -----------------------------
		// 2️⃣ 服务级流量统计
		// -----------------------------
		// 按服务名获取对应的流量计数器
		serviceCounter, err := public.FlowCounterHandler.GetCounter(
			public.FlowServicePrefix + serviceDetail.Info.ServiceName,
		)
		if err != nil {
			// 获取服务级计数器失败，返回错误并中断请求
			middleware.ResponseError(c, 4001, err)
			c.Abort()
			return
		}

		// 当前服务请求数 +1
		serviceCounter.Increase()

		// 可选：按天统计（调试用）
		// dayServiceCount, _ := serviceCounter.GetDayData(time.Now())
		// fmt.Printf("serviceCounter qps:%v, dayCount:%v", serviceCounter.QPS, dayServiceCount)

		// 放行请求，进入后续中间件或业务逻辑
		c.Next()
	}
}
