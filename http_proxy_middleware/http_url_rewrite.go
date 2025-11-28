package http_proxy_middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go-gateway/dao"
	"go-gateway/middleware"
	"regexp"
	"strings"
)

// HTTPUrlRewriteMiddleware URL 重写中间件
// 功能：根据服务配置的 UrlRewrite 规则，对请求 URL.Path 进行正则替换重写
// 典型用途：
// - 统一 API 前缀
// - 老接口兼容新接口
// - 灰度路由切换
func HTTPUrlRewriteMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 从上下文中获取当前请求对应的服务信息
		serverInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}

		// 类型断言为 ServiceDetail
		serviceDetail := serverInterface.(*dao.ServiceDetail)

		// UrlRewrite 配置格式示例：
		// "^/api/v1 /api/v2,^/old /new"
		// 多条规则通过逗号分隔
		for _, item := range strings.Split(serviceDetail.HTTPRule.UrlRewrite, ",") {

			// 每条规则格式：<正则表达式> <替换路径>
			items := strings.Split(item, " ")
			if len(items) != 2 {
				// 规则非法，直接跳过
				continue
			}

			// 编译正则表达式
			reg, err := regexp.Compile(items[0])
			if err != nil {
				// 正则非法，忽略该条规则
				continue
			}

			// 按正则规则对当前请求路径进行替换
			replacePath := reg.ReplaceAll(
				[]byte(c.Request.URL.Path),
				[]byte(items[1]),
			)

			// 将替换后的路径重新写回请求
			c.Request.URL.Path = string(replacePath)
		}

		// 继续后续中间件或业务逻辑
		c.Next()
	}
}
