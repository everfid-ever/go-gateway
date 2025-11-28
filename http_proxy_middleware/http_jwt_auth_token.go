package http_proxy_middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go-gateway/dao"
	"go-gateway/middleware"
	"go-gateway/public"
	"strings"
)

// HTTPJwtAuthTokenMiddleware JWT 鉴权中间件
// 功能：
// 1. 从 Authorization 头中解析 Bearer Token
// 2. 解析 JWT，获取 AppID（Issuer）
// 3. 在系统 App 列表中查找对应租户
// 4. 将 App 信息写入 gin.Context 供后续限流、统计使用
// 5. 若服务开启 OpenAuth 且未匹配到合法 App，则拒绝请求
func HTTPJwtAuthTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 获取当前请求匹配的服务信息
		serverInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}
		serviceDetail := serverInterface.(*dao.ServiceDetail)

		// 从 Authorization Header 中提取 Bearer Token
		// 格式示例：Authorization: Bearer xxxxx.yyyyy.zzzzz
		token := strings.ReplaceAll(c.GetHeader("Authorization"), "Bearer ", "")

		// 标识是否成功匹配到合法 App
		appMatched := false

		if token != "" {
			// 解析 JWT Token
			claims, err := public.JwtDecode(token)
			if err != nil {
				// Token 非法或过期
				middleware.ResponseError(c, 2002, err)
				c.Abort()
				return
			}

			// 从系统中所有 App 列表中匹配 Issuer 对应的 AppID
			appList := dao.AppManagerHandler.GetAppList()
			for _, appInfo := range appList {
				if appInfo.AppID == claims.Issuer {
					// 将匹配到的 App 信息写入 Context
					c.Set("app", appInfo)
					appMatched = true
					break
				}
			}
		}

		// 若服务开启鉴权但未匹配到合法 App，则直接拒绝访问
		if serviceDetail.AccessControl.OpenAuth == 1 && !appMatched {
			middleware.ResponseError(c, 2003, errors.New("not match valid app"))
			c.Abort()
			return
		}

		// 鉴权通过，继续后续中间件
		c.Next()
	}
}
