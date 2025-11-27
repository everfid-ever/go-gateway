package http_proxy_middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go-gateway/dao"
	"go-gateway/middleware"
	"strings"
)

// HTTPHeaderTransferMiddleware 用于对请求头进行动态转换的中间件
// 功能：根据 service 中配置的 HeaderTransfor 规则，对 HTTP 请求头进行 add/edit/del 操作
func HTTPHeaderTransferMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 从 gin.Context 中获取当前请求绑定的服务信息
		serverInterface, ok := c.Get("service")
		if !ok {
			// 未找到服务信息，直接返回错误并中断请求
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}

		// 类型断言为 ServiceDetail 结构体
		serviceDetail := serverInterface.(*dao.ServiceDetail)

		// HeaderTransfor 配置格式示例：
		// "add X-Token abc123,del X-Debug,edit X-Version v2"
		// 以逗号分隔多条规则
		for _, item := range strings.Split(serviceDetail.HTTPRule.HeaderTransfor, ",") {

			// 每条规则内部以空格分隔：操作 类型、Header Key、Header Value
			// 格式必须为：<op> <key> <value>
			items := strings.Split(item, " ")

			// 非法规则直接跳过，避免 panic
			if len(items) != 3 {
				continue
			}

			// add / edit 操作：设置或覆盖请求头
			if items[0] == "add" || items[0] == "edit" {
				c.Request.Header.Set(items[1], items[2])
			}

			// del 操作：删除指定请求头
			if items[0] == "del" {
				c.Request.Header.Del(items[1])
			}
		}

		// 放行请求，进入下一个中间件或业务处理逻辑
		c.Next()
	}
}
