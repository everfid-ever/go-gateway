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

// HTTPBlackListMiddleware IP 黑名单控制中间件
// 功能：根据服务的 AccessControl 配置，对客户端 IP 进行黑名单拦截
// 规则说明：
// 1. 仅当 OpenAuth == 1 时才生效
// 2. 当白名单为空、黑名单不为空时启用黑名单校验
// 3. 若客户端 IP 命中黑名单，则直接拒绝访问
func HTTPBlackListMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 从上下文中获取当前请求对应的服务信息
		serverInterface, ok := c.Get("service")
		if !ok {
			// 未获取到服务信息，返回错误并终止请求
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}

		// 类型断言，获取服务配置详情
		serviceDetail := serverInterface.(*dao.ServiceDetail)

		// -----------------------------
		// 解析白名单 IP 列表
		// -----------------------------
		whileIpList := []string{}
		if serviceDetail.AccessControl.WhiteList != "" {
			// 多个 IP 使用逗号分隔
			whileIpList = strings.Split(serviceDetail.AccessControl.WhiteList, ",")
		}

		// -----------------------------
		// 解析黑名单 IP 列表
		// -----------------------------
		blackIpList := []string{}
		if serviceDetail.AccessControl.BlackList != "" {
			// 多个 IP 使用逗号分隔
			blackIpList = strings.Split(serviceDetail.AccessControl.BlackList, ",")
		}

		// -----------------------------
		// 黑名单生效条件：
		// 1. 开启访问控制（OpenAuth == 1）
		// 2. 未配置白名单
		// 3. 已配置黑名单
		// -----------------------------
		if serviceDetail.AccessControl.OpenAuth == 1 &&
			len(whileIpList) == 0 &&
			len(blackIpList) > 0 {

			// 判断客户端 IP 是否在黑名单中
			if public.InStringSlice(blackIpList, c.ClientIP()) {

				// 命中黑名单，直接拒绝访问
				middleware.ResponseError(
					c,
					3001,
					errors.New(fmt.Sprintf("%s in black ip list", c.ClientIP())),
				)
				c.Abort()
				return
			}
		}

		// 未命中黑名单，继续后续中间件或业务逻辑
		c.Next()
	}
}
