package common

import (
	"os"
	"strconv"
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

// TokenFactoryRouteErrorThreshold 黏性渠道连续报错多少次后熔断切换（默认 3）。
// 环境变量 TOKENFACTORY_ROUTE_ERROR_THRESHOLD。
func TokenFactoryRouteErrorThreshold() int {
	if v, err := strconv.Atoi(os.Getenv("TOKENFACTORY_ROUTE_ERROR_THRESHOLD")); err == nil && v > 0 {
		return v
	}
	return 3
}

// TokenFactoryRouteStickyTTLSeconds 黏性绑定的存活秒数（默认 1800，即 30 分钟）。
// 环境变量 TOKENFACTORY_ROUTE_STICKY_TTL_SECONDS。
func TokenFactoryRouteStickyTTLSeconds() int {
	if v, err := strconv.Atoi(os.Getenv("TOKENFACTORY_ROUTE_STICKY_TTL_SECONDS")); err == nil && v > 0 {
		return v
	}
	return 1800
}

// TokenFactoryRouteErrorTTLSeconds 错误计数的窗口秒数（默认 600，超时即视为非「连续」）。
// 环境变量 TOKENFACTORY_ROUTE_ERROR_TTL_SECONDS。
func TokenFactoryRouteErrorTTLSeconds() int {
	if v, err := strconv.Atoi(os.Getenv("TOKENFACTORY_ROUTE_ERROR_TTL_SECONDS")); err == nil && v > 0 {
		return v
	}
	return 600
}

// TokenFactorySiteKey 返回当前站点的唯一标识，用于多站点隔离。
// 环境变量 TOKENFACTORY_SITE_KEY，如 "site-a"、"site-b"。
// 若为空，默认 "default"。
func TokenFactorySiteKey() string {
	key := os.Getenv("TOKENFACTORY_SITE_KEY")
	if key == "" {
		return "default"
	}
	return key
}
