package dto

import (
	"github.com/gin-gonic/gin"
	"go-gateway/public"
	"time"
)

// AdminSessionInfo 管理员登录后存入 session 的信息
type AdminSessionInfo struct {
	ID        int       `json:"id"`
	UserName  string    `json:"user_name"`
	LoginTime time.Time `json:"login_time"`
}

// AdminLoginInput 管理员登录请求参数
type AdminLoginInput struct {
	UserName string `json:"username" form:"username" comment:"管理员用户名" example:"admin" validate:"required,valid_username"` // 管理员用户名
	Password string `json:"password" form:"password" comment:"密码" example:"123456" validate:"required"`                   // 密码
}

// BindValidParam 绑定并校验登录参数
func (param *AdminLoginInput) BindValidParam(c *gin.Context) error {
	return public.DefaultGetValidParams(c, param)
}

// AdminLoginOutput 登录成功返回的 token
type AdminLoginOutput struct {
	Token string `json:"token" form:"token" comment:"token" example:"token" validate:""` // token
}
