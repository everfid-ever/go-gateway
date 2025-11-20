package controller

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-gateway/common/lib"
	"go-gateway/dao"
	"go-gateway/dto"
	"go-gateway/middleware"
	"go-gateway/public"
)

// AdminLoginController 管理员登录控制器
type AdminLoginController struct{}

// AdminLoginRegister 注册管理员登录路由
func AdminLoginRegister(group *gin.RouterGroup) {
	adminLogin := &AdminLoginController{}
	group.POST("/login", adminLogin.AdminLogin)
}

// AdminLogin godoc
// @Summary 管理员登陆
// @Description 管理员登陆
// @Tags 管理员接口
// @ID /admin_login/login
// @Accept  json
// @Produce  json
// @Param body body dto.AdminLoginInput true "body"
// @Success 200 {object} middleware.Response{data=dto.AdminLoginOutput} "success"
// @Router /admin_login/login [post]
// 管理员登录处理：参数校验、账号校验、写入 session、返回 token
func (adminlogin *AdminLoginController) AdminLogin(c *gin.Context) {
	// 解析并校验参数
	params := &dto.AdminLoginInput{}
	if err := params.BindValidParam(c); err != nil {
		middleware.ResponseError(c, 2000, err)
		return
	}

	//1. params.UserName 取得管理员信息 admininfo
	//2. admininfo.salt + params.Password sha256 => saltPassword
	//3. saltPassword==admininfo.password
	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, 2001, err)
		return
	}

	// 用户名密码校验
	admin := &dao.Admin{}
	admin, err = admin.LoginCheck(c, tx, params)
	if err != nil {
		middleware.ResponseError(c, 2002, err)
		return
	}

	// 组装并写入 session
	sessInfo := &dto.AdminSessionInfo{
		ID:        admin.Id,
		UserName:  admin.UserName,
		LoginTime: time.Now(),
	}
	sessBts, err := json.Marshal(sessInfo)
	if err != nil {
		middleware.ResponseError(c, 2003, err)
		return
	}
	sess := sessions.Default(c)
	sess.Set(public.AdminSessionInfoKey, string(sessBts))
	if err := sess.Save(); err != nil { // 保存失败直接返回
		middleware.ResponseError(c, 2004, err)
		return
	}

	// 返回结果（这里用用户名作为简单 token）
	out := &dto.AdminLoginOutput{Token: admin.UserName}
	middleware.ResponseSuccess(c, out)
}
