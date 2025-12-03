package grpc_proxy_middleware

import (
	"github.com/pkg/errors"
	"go-gateway/dao"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"log"
	"strings"
)

// GrpcHeaderTransferMiddleware 创建一个 gRPC 流式调用的“Header 透传 / 修改中间件”
// 作用：
//
//	根据 serviceDetail.GRPCRule.HeaderTransfor 中的配置规则，
//	对当前 gRPC 请求中的 Metadata（Header）进行：
//	- 新增(add)
//	- 修改(edit)
//	- 删除(del)
//
// 然后将处理后的 Metadata 重新设置回当前 Stream 中，
// 再继续执行后续的业务处理逻辑。
func GrpcHeaderTransferMiddleware(serviceDetail *dao.ServiceDetail) func(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {

	// 返回真正被 gRPC Server 执行的 Stream 拦截函数
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {

		// -----------------------------
		// 1. 从当前 gRPC 上下文中获取请求的 Metadata（即 Header）
		// -----------------------------
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			// 如果上下文中未携带 Metadata，说明请求不完整或异常
			return errors.New("miss metadata from context")
		}

		// -----------------------------
		// 2. 解析配置的 Header 转换规则
		// 规则格式示例（用逗号分隔多条规则）：
		//   "add x-user-id 1001,edit x-app-id app1,del x-debug"
		//
		// 每条规则被空格拆分为 3 段：
		//   items[0] → 操作类型（add / edit / del）
		//   items[1] → Header Key
		//   items[2] → Header Value（del 时可忽略）
		// -----------------------------
		for _, item := range strings.Split(
			serviceDetail.GRPCRule.HeaderTransfor,
			",",
		) {

			items := strings.Split(item, " ")

			// 非法规则直接跳过，防止数组越界
			if len(items) != 3 {
				continue
			}

			// -----------------------------
			// 3. 根据规则类型对 Metadata 进行操作
			// -----------------------------

			// add / edit：设置或覆盖指定 Header
			if items[0] == "add" || items[0] == "edit" {
				// items[1] 为 key，items[2] 为 value
				md.Set(items[1], items[2])
			}

			// del：删除指定 Header
			if items[0] == "del" {
				delete(md, items[1])
			}
		}

		// -----------------------------
		// 4. 将修改后的 Metadata 重新写回 gRPC Stream Header
		// -----------------------------
		if err := ss.SetHeader(md); err != nil {
			// 如果设置 Header 失败，直接返回带上下文信息的错误
			return errors.WithMessage(err, "SetHeader")
		}

		// -----------------------------
		// 5. Header 处理完成，继续执行真正的 gRPC 业务处理逻辑
		// -----------------------------
		if err := handler(srv, ss); err != nil {
			// 业务处理失败，记录日志并返回错误
			log.Printf("RPC failed with error %v\n", err)
			return err
		}

		// -----------------------------
		// 6. 正常执行完成
		// -----------------------------
		return nil
	}
}
