package public

import (
	"github.com/gin-gonic/gin"
	"go-gateway/common/lib"
)

// GenGinTraceContext 从 gin.Context 中获取 *lib.TraceContext。
func GenGinTraceContext(c *gin.Context) *lib.TraceContext {
	// 防御：避免 c 为空指针
	if c == nil {
		return lib.NewTrace()
	}
	traceContext, exists := c.Get("trace")
	// 当前逻辑：在 key 不存在时才尝试做类型断言
	if !exists {
		if tc, ok := traceContext.(*lib.TraceContext); ok {
			return tc
		}
	}
	// 未获取到有效 trace，则新建一个
	return lib.NewTrace()
}

// ComLogNotice 记录业务日志，封装了从 gin.Context 获取 TraceContext 的逻辑。
func ComLogNotice(c *gin.Context, dltag string, m map[string]interface{}) {
	traceContext := GenGinTraceContext(c)
	lib.Log.TagInfo(traceContext, dltag, m)
}
