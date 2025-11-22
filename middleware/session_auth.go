package middleware

import (
	"errors"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-gateway/public"
)

// SessionAuthMiddleware 校验管理员是否已登录的中间件。
// 通过读取 Session 中的 AdminSessionInfoKey 判断登录态是否存在，
// 若未登录则直接返回错误并中断请求链路。
func SessionAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)

		// 从 session 中读取登录信息，不存在则说明未登录
		if adminInfo, ok := session.Get(public.AdminSessionInfoKey).(string); !ok || adminInfo == "" {
			ResponseError(c, InternalErrorCode, errors.New("user not login"))
			// 终止后续处理
			c.Abort()
			return
		}
		c.Next()
	}
}
