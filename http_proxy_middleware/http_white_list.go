package http_proxy_middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go-gateway/dao"
	"go-gateway/middleware"
	"go-gateway/public"
	"strings"
)

// HTTPWhiteListMiddleware IP 白名单控制中间件
// 功能：根据服务的 AccessControl 配置，对客户端 IP 进行白名单校验
// 规则说明：
// 1. 仅当 OpenAuth == 1 时白名单控制才生效
// 2. 当白名单列表不为空时，必须命中白名单才允许访问
// 3. 未命中白名单的请求将被直接拦截
func HTTPWhiteListMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 从上下文中获取当前请求所对应的服务信息
		serverInterface, ok := c.Get("service")
		if !ok {
			// 未找到服务信息，返回错误并中断请求
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}

		// 类型断言，获取完整的服务配置
		serviceDetail := serverInterface.(*dao.ServiceDetail)

		// -----------------------------
		// 解析白名单 IP 列表
		// -----------------------------
		iplist := []string{}
		if serviceDetail.AccessControl.WhiteList != "" {
			// 多个 IP 使用逗号分隔
			iplist = strings.Split(serviceDetail.AccessControl.WhiteList, ",")
		}

		// -----------------------------
		// 白名单校验规则：
		// 1. 开启访问控制（OpenAuth == 1）
		// 2. 白名单不为空
		// 3. 当前客户端 IP 必须存在于白名单中
		// -----------------------------
		if serviceDetail.AccessControl.OpenAuth == 1 && len(iplist) > 0 {

			// 当前客户端 IP 不在白名单内，拒绝访问
			if !public.InStringSlice(iplist, c.ClientIP()) {
				middleware.ResponseError(
					c,
					3001,
					errors.New(fmt.Sprintf("%s not in white ip list", c.ClientIP())),
				)
				c.Abort()
				return
			}
		}

		// 通过白名单校验，放行请求
		c.Next()
	}
}
