package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go-gateway/common/lib"
	"go-gateway/dao"
	"go-gateway/dto"
	"go-gateway/middleware"
	"go-gateway/public"
	"time"
)

// DashboardController 首页大盘控制器
type DashboardController struct{}

// DashboardRegister 注册路由
func DashboardRegister(group *gin.RouterGroup) {
	service := &DashboardController{}
	group.GET("/panel_group_data", service.PanelGroupData) // 指标统计接口
	group.GET("/flow_stat", service.FlowStat)              // 当天/昨天流量趋势
	group.GET("/service_stat", service.ServiceStat)        // 服务类型占比统计
}

// PanelGroupData godoc
// @Summary 指标统计
// @Description 指标统计
// @Tags 首页大盘
// @ID /dashboard/panel_group_data
// @Accept  json
// @Produce  json
// @Success 200 {object} middleware.Response{data=dto.PanelGroupDataOutput} "success"
// @Router /dashboard/panel_group_data [get]
func (service *DashboardController) PanelGroupData(c *gin.Context) {

	// 1. 获取 GORM 默认数据库连接
	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, 2001, err)
		return
	}

	// 2. 查询服务数量（使用 PageList 查询第一页，但只要总数）
	serviceInfo := &dao.ServiceInfo{}
	_, serviceNum, err := serviceInfo.PageList(
		c,
		tx,
		&dto.ServiceListInput{PageSize: 1, PageNo: 1},
	)
	if err != nil {
		middleware.ResponseError(c, 2002, err)
		return
	}

	// 3. 查询 APP 数量（同样用分页接口，只取 total 数）
	app := &dao.App{}
	_, appNum, err := app.APPList(
		c,
		tx,
		&dto.APPListInput{PageNo: 1, PageSize: 1},
	)
	if err != nil {
		middleware.ResponseError(c, 2002, err)
		return
	}

	// 4. 获取总流量计数器（流量统计模块）
	counter, err := public.FlowCounterHandler.GetCounter(public.FlowTotal)
	if err != nil {
		middleware.ResponseError(c, 2003, err)
		return
	}

	// 5. 组装返回数据
	out := &dto.PanelGroupDataOutput{
		ServiceNum:      serviceNum,         // 服务数量
		AppNum:          appNum,             // APP 数量
		TodayRequestNum: counter.TotalCount, // 今日总请求数
		CurrentQPS:      counter.QPS,        // 当前 QPS
	}

	// 6. 返回前端
	middleware.ResponseSuccess(c, out)
}

// ServiceStat godoc
// @Summary 服务统计
// @Description 服务统计
// @Tags 首页大盘
// @ID /dashboard/service_stat
// @Accept  json
// @Produce  json
// @Success 200 {object} middleware.Response{data=dto.DashServiceStatOutput} "success"
// @Router /dashboard/service_stat [get]
func (service *DashboardController) ServiceStat(c *gin.Context) {

	// 1. 获取数据库连接
	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, 2001, err)
		return
	}

	// 2. 按 LoadType 分组统计：HTTP/TCP/GRPC 各多少个服务
	serviceInfo := &dao.ServiceInfo{}
	list, err := serviceInfo.GroupByLoadType(c, tx)
	if err != nil {
		middleware.ResponseError(c, 2002, err)
		return
	}

	// 3. 构建 legend（前端图表用）
	legend := []string{}
	for index, item := range list {
		name, ok := public.LoadTypeMap[item.LoadType]
		if !ok {
			middleware.ResponseError(c, 2003, errors.New("load_type not found"))
			return
		}
		list[index].Name = name // 图表显示名称
		legend = append(legend, name)
	}

	// 4. 返回前端
	out := &dto.DashServiceStatOutput{
		Legend: legend, // ["HTTP", "TCP", "GRPC"]
		Data:   list,   // 每种 loadType 的数量
	}
	middleware.ResponseSuccess(c, out)
}

// FlowStat godoc
// @Summary 服务统计
// @Description 服务统计
// @Tags 首页大盘
// @ID /dashboard/flow_stat
// @Accept  json
// @Produce  json
// @Success 200 {object} middleware.Response{data=dto.ServiceStatOutput} "success"
// @Router /dashboard/flow_stat [get]
func (service *DashboardController) FlowStat(c *gin.Context) {

	// 1. 获取总流量计数器
	counter, err := public.FlowCounterHandler.GetCounter(public.FlowTotal)
	if err != nil {
		middleware.ResponseError(c, 2001, err)
		return
	}

	// 2. 生成今日每小时请求量数组
	todayList := []int64{}
	currentTime := time.Now()
	for i := 0; i <= currentTime.Hour(); i++ { // 比如现在下午 3 点，只取 0~15 点的数据
		dateTime := time.Date(
			currentTime.Year(),
			currentTime.Month(),
			currentTime.Day(),
			i, 0, 0, 0,
			lib.TimeLocation,
		)
		hourData, _ := counter.GetHourData(dateTime)
		todayList = append(todayList, hourData)
	}

	// 3. 生成昨日整天 24 小时请求量
	yesterdayList := []int64{}
	yesterTime := currentTime.Add(-24 * time.Hour)
	for i := 0; i <= 23; i++ {
		dateTime := time.Date(
			yesterTime.Year(),
			yesterTime.Month(),
			yesterTime.Day(),
			i, 0, 0, 0,
			lib.TimeLocation,
		)
		hourData, _ := counter.GetHourData(dateTime)
		yesterdayList = append(yesterdayList, hourData)
	}

	// 4. 返回今日 vs 昨日
	middleware.ResponseSuccess(c, &dto.ServiceStatOutput{
		Today:     todayList,
		Yesterday: yesterdayList,
	})
}
