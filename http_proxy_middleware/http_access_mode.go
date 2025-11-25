package http_proxy_middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-gateway/dao"
	"go-gateway/middleware"
	"go-gateway/public"
)

// HTTPAccessModeMiddleware 是一个基于 HTTP 请求信息匹配服务接入方式的中间件。
// 主要职责：
// 1. 从请求中提取域名、路径等信息，用于匹配对应的服务配置（HTTP 接入方式）。
// 2. 若匹配失败，立即返回错误响应并中断后续处理。
// 3. 若匹配成功，将匹配到的 ServiceDetail 存入 Gin Context，供后续处理链使用。
//
// 使用场景：
// - 网关代理层根据域名/前缀分发请求
// - 为不同服务做访问控制、负载均衡、限流、熔断等策略前置判断
//
// 上下文关键字：
// - "service"：匹配到的服务配置对象，可在后续中间件和 handler 中通过 c.Get("service") 获取
func HTTPAccessModeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 基于请求信息（域名、路径等）匹配对应的 ServiceDetail
		service, err := dao.ServiceManagerHandler.HTTPAccessMode(c)
		if err != nil {
			// 若匹配失败，返回错误并终止请求
			middleware.ResponseError(c, 1001, err)
			c.Abort()
			return
		}

		// 打印调试信息：当前匹配到的 service 配置
		fmt.Println("matched service", public.Obj2Json(service))

		// 将服务配置写入 Context，供后续处理中间件或 handler 使用
		c.Set("service", service)

		// 继续执行下一个中间件/handler
		c.Next()
	}
}
