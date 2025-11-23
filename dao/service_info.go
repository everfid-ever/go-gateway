package dao

import (
	"github.com/e421083458/gorm"
	"github.com/gin-gonic/gin"
	"go-gateway/dto"
	"go-gateway/public"
	"time"
)

type ServiceInfo struct {
	ID          int64     `json:"id" gorm:"primary_key"`
	LoadType    int       `json:"load_type" gorm:"column:load_type" description:"负载类型 0=http 1=tcp 2=grpc"`
	ServiceName string    `json:"service_name" gorm:"column:service_name" description:"服务名称"`
	ServiceDesc string    `json:"service_desc" gorm:"column:service_desc" description:"服务描述"`
	UpdatedAt   time.Time `json:"create_at" gorm:"column:create_at" description:"更新时间"`
	CreatedAt   time.Time `json:"update_at" gorm:"column:update_at" description:"添加时间"`
	IsDelete    int8      `json:"is_delete" gorm:"column:is_delete" description:"是否已删除；0：否；1：是"`
}

func (t *ServiceInfo) TableName() string {
	return "gateway_service_info"
}

// ServiceDetail 读取某个服务的完整配置信息。
// 包含：基础信息、HTTP/TCP/GRPC 规则、负载均衡、访问控制等。
// search 可只填 ServiceID 或 ServiceName，不完整时会自动补全。
func (t *ServiceInfo) ServiceDetail(c *gin.Context, tx *gorm.DB, search *ServiceInfo) (*ServiceDetail, error) {

	// 若 search 中只有部分字段（例如只有 service_id），
	// 则从数据库补全完整的 ServiceInfo。
	if search.ServiceName == "" {
		info, err := t.Find(c, tx, search)
		if err != nil {
			return nil, err
		}
		search = info
	}

	// 读取 HTTP 规则（可选）
	httpRule := &HttpRule{ServiceID: search.ID}
	httpRule, err := httpRule.Find(c, tx, httpRule)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 读取 TCP 规则（可选）
	tcpRule := &TcpRule{ServiceID: search.ID}
	tcpRule, err = tcpRule.Find(c, tx, tcpRule)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 读取 GRPC 规则（可选）
	grpcRule := &GrpcRule{ServiceID: search.ID}
	grpcRule, err = grpcRule.Find(c, tx, grpcRule)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 读取访问控制配置（可选）
	accessControl := &AccessControl{ServiceID: search.ID}
	accessControl, err = accessControl.Find(c, tx, accessControl)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 读取负载均衡策略（可选）
	loadBalance := &LoadBalance{ServiceID: search.ID}
	loadBalance, err = loadBalance.Find(c, tx, loadBalance)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 聚合所有配置，形成完整服务详情
	detail := &ServiceDetail{
		Info:          search,
		HTTPRule:      httpRule,
		TCPRule:       tcpRule,
		GRPCRule:      grpcRule,
		LoadBalance:   loadBalance,
		AccessControl: accessControl,
	}

	return detail, nil
}

func (t *ServiceInfo) GroupByLoadType(c *gin.Context, tx *gorm.DB) ([]dto.DashServiceStatItemOutput, error) {
	list := []dto.DashServiceStatItemOutput{}
	query := tx.SetCtx(public.GetGinTraceContext(c))
	if err := query.Table(t.TableName()).Where("is_delete=0").Select("load_type, count(*) as value").Group("load_type").Scan(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// PageList 分页查询服务列表。
// 支持按 service_name / service_desc 模糊搜索，并返回列表和总数。
func (t *ServiceInfo) PageList(c *gin.Context, tx *gorm.DB, param *dto.ServiceListInput) ([]ServiceInfo, int64, error) {
	total := int64(0)
	list := []ServiceInfo{}
	offset := (param.PageNo - 1) * param.PageSize

	// 基础查询：未删除的数据 + 链路追踪上下文
	query := tx.SetCtx(public.GetGinTraceContext(c)).
		Table(t.TableName()).
		Where("is_delete=0")

	// 模糊搜索：service_name / service_desc
	if param.Info != "" {
		query = query.Where("(service_name like ? OR service_desc like ?)",
			"%"+param.Info+"%", "%"+param.Info+"%")
	}

	// 分页 + 排序查询列表
	if err := query.
		Limit(param.PageSize).
		Offset(offset).
		Order("id desc").
		Find(&list).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, 0, err
	}

	// 查询总数（再次使用同一 query）
	query.Count(&total)

	return list, total, nil
}

// Find 根据结构体字段精准查询一条记录。
// search 结构体中非零值字段将作为查询条件
func (t *ServiceInfo) Find(c *gin.Context, tx *gorm.DB, search *ServiceInfo) (*ServiceInfo, error) {
	out := &ServiceInfo{}
	err := tx.SetCtx(public.GetGinTraceContext(c)).Where(search).Find(out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (t *ServiceInfo) Save(c *gin.Context, tx *gorm.DB) error {
	return tx.SetCtx(public.GetGinTraceContext(c)).Save(t).Error
}
