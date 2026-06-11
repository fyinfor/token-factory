package common

import (
	"os"
)

// ── TokenFactory gRPC 连接配置 ──────────────────────────────────

// TokenFactoryGRPCEndpoint 返回 TokenFactory gRPC 服务地址。
// 环境变量 TOKENFACTORY_GRPC_ENDPOINT，默认 ":9000"。
func TokenFactoryGRPCEndpoint() string {
	ep := os.Getenv("TOKENFACTORY_GRPC_ENDPOINT")
	if ep == "" {
		return ":9000"
	}
	return ep
}

// TokenFactoryRouteEnabled 是否启用 TokenFactory 智能路由。
// 环境变量 TOKENFACTORY_ROUTE_ENABLED=true 时启用。
func TokenFactoryRouteEnabled() bool {
	return os.Getenv("TOKENFACTORY_ROUTE_ENABLED") == "true"
}

// TokenFactorySyncSecret 返回 TFOpenSync 同步密钥。
func TokenFactorySyncSecret() string {
	return os.Getenv("TOKENFACTORY_OPEN_SYNC_SECRET")
}
