package http_proxy_middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go-gateway/dao"
	"go-gateway/middleware"
	"go-gateway/public"
	"strings"
)

func HTTPStripUriMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取服务信息
		serverInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}
		serviceDetail := serverInterface.(*dao.ServiceDetail)

		// 判断是否需要剥离URI
		if serviceDetail.HTTPRule.RuleType == public.HTTPRuleTypePrefixURL && serviceDetail.HTTPRule.NeedStripUri == 1 {
			//fmt.Println("c.Request.URL.Path",c.Request.URL.Path)
			// 剥离URI前缀
			c.Request.URL.Path = strings.Replace(c.Request.URL.Path, serviceDetail.HTTPRule.Rule, "", 1)
			//fmt.Println("c.Request.URL.Path",c.Request.URL.Path)
		}
		//http://127.0.0.1:8080/test_http_string/abbb
		//http://127.0.0.1:2004/abbb

		c.Next()
	}
}
