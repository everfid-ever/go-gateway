package grpc_proxy_middleware

import (
	"go-gateway/dao"
	"go-gateway/public"
	"google.golang.org/grpc"
	"log"
)

// GrpcFlowCountMiddleware 创建一个 gRPC 流式调用的“流量统计中间件”
// 作用：
//
//	在每一次 gRPC Stream 请求进入业务处理前，
//	分别对：
//	  1) 系统全局总请求量
//	  2) 当前服务的请求量
//	进行累计统计，用于监控、限流、报表分析等场景。
func GrpcFlowCountMiddleware(serviceDetail *dao.ServiceDetail) func(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {

	// 返回真正被 gRPC Server 注册执行的 Stream 拦截函数
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {

		// -----------------------------
		// 1. 获取“系统全局总流量计数器”
		// public.FlowTotal：用于标识全局流量统计的 Key
		// -----------------------------
		totalCounter, err := public.FlowCounterHandler.GetCounter(
			public.FlowTotal,
		)
		if err != nil {
			// 如果计数器获取失败，直接返回错误，阻止请求继续执行
			return err
		}

		// 对全局总请求数进行一次递增
		totalCounter.Increase()

		// -----------------------------
		// 2. 获取“当前服务级流量计数器”
		// public.FlowServicePrefix：服务级流量统计前缀
		// serviceDetail.Info.ServiceName：当前服务名称
		// -----------------------------
		serviceCounter, err := public.FlowCounterHandler.GetCounter(
			public.FlowServicePrefix + serviceDetail.Info.ServiceName,
		)
		if err != nil {
			// 获取服务级计数器失败，同样直接返回错误
			return err
		}

		// 对当前服务的请求量进行递增
		serviceCounter.Increase()

		// -----------------------------
		// 3. 流量统计完成后，继续执行真正的 gRPC 业务处理逻辑
		// -----------------------------
		if err := handler(srv, ss); err != nil {
			// 业务处理失败，记录日志并返回错误
			log.Printf(
				"GrpcFlowCountMiddleware failed with error %v\n",
				err,
			)
			return err
		}

		// -----------------------------
		// 4. 正常执行完成，返回成功
		// -----------------------------
		return nil
	}
}
