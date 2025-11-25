package http_proxy_middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go-gateway/dao"
	"go-gateway/middleware"
	"go-gateway/reverse_proxy"
)

// HTTPReverseProxyMiddleware 是 HTTP 反向代理中间件。
// 该中间件在完成服务匹配、限流、权限等前序步骤后执行，
// 负责：
//  1. 从上下文中读取已匹配的服务信息（ServiceDetail）。
//  2. 根据服务配置选择对应的负载均衡器实例。
//  3. 根据服务配置创建或获取 HTTP 传输代理（Transport）。
//  4. 创建基于负载均衡算法的 ReverseProxy。
//  5. 将当前请求转发到后端服务节点，并直接返回响应。
//
// 注意：
//   - ReverseProxy 直接对 c.Writer 写响应，因此本中间件执行后需要 Abort，
//     防止 Gin 继续向下执行其他 Handler。
//   - 若服务、负载均衡、传输器任意一步失败，则直接返回错误响应。
func HTTPReverseProxyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 1. 从 Gin Context 中读取匹配的服务详情
		serverInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}
		serviceDetail := serverInterface.(*dao.ServiceDetail)

		// 2. 获取负载均衡器（随机、轮询、加权等策略）
		lb, err := dao.LoadBalancerHandler.GetLoadBalancer(serviceDetail)
		if err != nil {
			middleware.ResponseError(c, 2002, err)
			c.Abort()
			return
		}

		// 3. 获取服务的 Transport（HTTP Client 代理）用于转发请求
		trans, err := dao.TransportorHandler.GetTrans(serviceDetail)
		if err != nil {
			middleware.ResponseError(c, 2003, err)
			c.Abort()
			return
		}

		// 4. 创建一个基于负载均衡的反向代理，并将请求转发到后端服务
		proxy := reverse_proxy.NewLoadBalanceReverseProxy(c, lb, trans)

		// proxy 会直接写响应，因此这里不再调用 c.Next()
		proxy.ServeHTTP(c.Writer, c.Request)

		// 5. 终止后续中间件，防止重复写响应
		c.Abort()
		return
	}
}
