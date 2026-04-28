// Package service: channel_health 维护 (channel_id, model_name) 的实时健康度（EWMA 时延、滑窗成功率、熔断状态机），
// 供 PR3+ 的 smart_router / distributor 直接消费。本文件本身不依赖 routing_policies；阈值用环境变量+常量兜底，
// 后续 PR2 引入策略表后再把阈值替换为按 policy 读取，但下游 API 形态不变。
//
// 核心契约：
//   - RecordChannelHealthSignal 是「读旧行 → 计算新指标 → Upsert」的一体化入口；调用方只需告诉发生了什么。
//   - 滑窗采用「指数衰减计数」近似：sample_count、success_count 各按 exp(-Δt/halfLife*ln2) 衰减后再 +1，
//     避免落库历史样本表，所有状态都在一行里推进，便于跨实例直接读 DB。
//   - 熔断走 closed → open（达到阈值）→ half_open（冷却到期，等待探针）→ closed/open 三态。
//
// 调用约定：
//   - relay 完成后（成功或失败）调用一次；
//   - 渠道单测 / 上架批量测试也可调用，传 Source = HealthSignalSourceManual / Onboard / Scheduled。
package service

import (
	"context"
	"errors"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
)

const (
	// channelHealthEwmaAlpha EWMA 平滑系数 α；越大越敏感（更快跟随最新值），LiteLLM 默认 0.2。
	channelHealthEwmaAlpha = 0.2
	// channelHealthSampleHalfLifeSeconds 滑窗"半衰期"：每过该秒数，旧样本权重衰减一半。
	channelHealthSampleHalfLifeSeconds = 60.0
	// channelHealthMinRequestsForCircuit 触发熔断所需的最小样本数（避免冷启动单次失败被开断路）。
	channelHealthMinRequestsForCircuit = 20
	// channelHealthFailureRateThreshold 失败率阈值，超过即开断路。
	channelHealthFailureRateThreshold = 0.5
	// channelHealthCooldownSeconds 熔断冷却基础时长；连续触发会指数退避到上限。
	channelHealthCooldownSeconds = 30
	// channelHealthCooldownMaxSeconds 熔断冷却时长上限。
	channelHealthCooldownMaxSeconds = 600
	// channelHealthCooldownAuth 鉴权 / 4xx 类立刻拉长冷却时长（这类错误很难自愈）。
	channelHealthCooldownAuthSeconds = 300
)

// channelHealthLocks 给 (channel_id, model_name) 一把粒度锁，避免并发请求把熔断状态算花。
// 单实例并发安全；多实例下因状态都落库，只是同一行可能被并发覆盖，不破坏长期收敛。
var channelHealthLocks sync.Map

func channelHealthMutex(channelID int, modelName string) *sync.Mutex {
	key := strconv.Itoa(channelID) + "|" + strings.TrimSpace(modelName)
	if v, ok := channelHealthLocks.Load(key); ok {
		return v.(*sync.Mutex)
	}
	mu := &sync.Mutex{}
	actual, _ := channelHealthLocks.LoadOrStore(key, mu)
	return actual.(*sync.Mutex)
}

// HealthSignalInput 是一次健康度采样：调用方按实际情况填字段，nil/0 由本文件按规则补默认。
//   - LatencyMs：必填；TtftMs、ThroughputTps 仅在采样到时填。
//   - HTTPStatus / TokenFactoryError / RawErr：用于错误分类，三者择一即可（优先 TokenFactoryError）。
//   - EndpointType：例如 "chat" / "responses" / "embeddings"，用于区分同渠道多端点的指标。
type HealthSignalInput struct {
	ChannelID         int
	ModelName         string
	Success           bool
	LatencyMs         int64
	TtftMs            int64
	ThroughputTps     float64
	HTTPStatus        int
	TokenFactoryError *types.TokenFactoryError
	RawErr            error
	EndpointType      string
	Message           string
	Source            string
}

// RecordChannelHealthSignal 写入一次健康度采样并维护熔断状态机；线程安全。
// 失败时仅记录日志、不向上抛错，避免拖累请求链路。
func RecordChannelHealthSignal(ctx context.Context, in HealthSignalInput) {
	in.ModelName = strings.TrimSpace(in.ModelName)
	if in.ChannelID <= 0 || in.ModelName == "" {
		return
	}
	if in.Source == "" {
		in.Source = model.HealthSignalSourceRelay
	}

	mu := channelHealthMutex(in.ChannelID, in.ModelName)
	mu.Lock()
	defer mu.Unlock()

	prev, err := model.GetModelTestResult(in.ChannelID, in.ModelName)
	if err != nil {
		common.SysError("RecordChannelHealthSignal: load row failed: " + err.Error())
		return
	}

	signal := composeHealthSignal(prev, in)
	if err := model.UpsertModelTestResultSignal(signal); err != nil {
		common.SysError("RecordChannelHealthSignal: upsert failed: " + err.Error())
	}

	_ = ctx
}

// composeHealthSignal 把前一行 + 本次采样合成最终的 HealthSignal（model 层入参）。
// 抽出来便于单测；不读 DB、纯函数，所有时间从 in 里推断（now 用 common.GetTimestamp）。
func composeHealthSignal(prev *model.ModelTestResult, in HealthSignalInput) model.HealthSignal {
	now := common.GetTimestamp()

	prevEwma := 0.0
	prevSampleCount := 0.0
	prevSuccessCount := 0.0
	prevConsecFail := 0
	prevState := model.CircuitStateClosed
	prevUnhealthyUntil := int64(0)
	prevLastTime := int64(0)
	if prev != nil {
		prevEwma = prev.LatencyEwmaMs
		prevSampleCount = float64(prev.RecentSampleCount)
		prevSuccessCount = prev.RecentSuccessRate * float64(prev.RecentSampleCount)
		prevConsecFail = prev.ConsecutiveFailCount
		if prev.CircuitState != "" {
			prevState = prev.CircuitState
		}
		prevUnhealthyUntil = prev.UnhealthyUntil
		prevLastTime = prev.LastTestTime
	}

	// 1) EWMA 时延：仅成功路径参与，失败请求耗时通常被超时/退避污染。
	newEwma := prevEwma
	if in.Success && in.LatencyMs > 0 {
		if prevEwma <= 0 {
			newEwma = float64(in.LatencyMs)
		} else {
			newEwma = channelHealthEwmaAlpha*float64(in.LatencyMs) + (1-channelHealthEwmaAlpha)*prevEwma
		}
	}

	// 2) 滑窗：按时间衰减旧计数（半衰期 channelHealthSampleHalfLifeSeconds）后再 +1。
	decay := 1.0
	if prevLastTime > 0 && now > prevLastTime {
		dt := float64(now - prevLastTime)
		// w = 0.5^(dt / halfLife)
		decay = math.Exp2(-dt / channelHealthSampleHalfLifeSeconds)
	}
	newSampleCount := prevSampleCount*decay + 1.0
	newSuccessCount := prevSuccessCount * decay
	if in.Success {
		newSuccessCount += 1.0
	}
	successRate := 0.0
	if newSampleCount > 0 {
		successRate = newSuccessCount / newSampleCount
	}

	// 3) 连续失败 / 熔断状态机。
	consecFail := prevConsecFail
	if in.Success {
		consecFail = 0
	} else {
		consecFail++
	}

	category := classifyError(in)

	state, unhealthyUntil := transitionCircuit(circuitInputs{
		now:               now,
		prevState:         prevState,
		prevUnhealthyTill: prevUnhealthyUntil,
		success:           in.Success,
		sampleCount:       newSampleCount,
		successRate:       successRate,
		category:          category,
	})

	// 4) 组装 HealthSignal。
	sampleCountInt := int(math.Round(newSampleCount))
	out := model.HealthSignal{
		ChannelID:            in.ChannelID,
		ModelName:            in.ModelName,
		Success:              in.Success,
		ResponseTimeMs:       in.LatencyMs,
		Message:              in.Message,
		Source:               in.Source,
		LatencyEwmaMs:        ptrFloat(newEwma),
		RecentSuccessRate:    ptrFloat(successRate),
		RecentSampleCount:    ptrInt(sampleCountInt),
		ConsecutiveFailCount: ptrInt(consecFail),
		UnhealthyUntil:       ptrInt64(unhealthyUntil),
		CircuitState:         ptrString(state),
	}
	if in.TtftMs > 0 {
		v := int(in.TtftMs)
		out.TtftMs = &v
	}
	if in.ThroughputTps > 0 {
		v := in.ThroughputTps
		out.ThroughputTps = &v
	}
	if in.EndpointType != "" {
		v := in.EndpointType
		out.LastTestEndpointType = &v
	}
	if !in.Success {
		v := category
		out.LastErrorCategory = &v
		if in.HTTPStatus > 0 {
			s := in.HTTPStatus
			out.LastHttpStatus = &s
		}
	} else {
		// 成功时清空错误类目与 HTTP 状态，避免老错误信息粘连。
		empty := model.ErrorCategoryNone
		out.LastErrorCategory = &empty
		zero := 0
		out.LastHttpStatus = &zero
	}
	return out
}

// circuitInputs 收敛 transitionCircuit 所需上下文，方便单测。
type circuitInputs struct {
	now               int64
	prevState         string
	prevUnhealthyTill int64
	success           bool
	sampleCount       float64
	successRate       float64
	category          string
}

// transitionCircuit 根据上一状态与本次采样推进熔断状态机；返回 (新状态, 新 unhealthy_until)。
//
//	closed   ──失败率超阈值──> open
//	open     ──冷却到期──>     half_open
//	half_open ──成功──>        closed
//	half_open ──失败──>        open(冷却时长翻倍, 上限 channelHealthCooldownMaxSeconds)
func transitionCircuit(in circuitInputs) (string, int64) {
	cooldown := int64(channelHealthCooldownSeconds)
	if in.category == model.ErrorCategoryAuth {
		cooldown = int64(channelHealthCooldownAuthSeconds)
	}

	switch in.prevState {
	case model.CircuitStateOpen:
		// 冷却未到期：保持 open。
		if in.prevUnhealthyTill > 0 && in.now < in.prevUnhealthyTill {
			return model.CircuitStateOpen, in.prevUnhealthyTill
		}
		// 冷却到期：转 half_open，等待第一笔探针结果，本次也算探针。
		if in.success {
			return model.CircuitStateClosed, 0
		}
		nextCooldown := cooldown * 2
		if nextCooldown > int64(channelHealthCooldownMaxSeconds) {
			nextCooldown = int64(channelHealthCooldownMaxSeconds)
		}
		return model.CircuitStateOpen, in.now + nextCooldown
	case model.CircuitStateHalfOpen:
		if in.success {
			return model.CircuitStateClosed, 0
		}
		nextCooldown := cooldown * 2
		if nextCooldown > int64(channelHealthCooldownMaxSeconds) {
			nextCooldown = int64(channelHealthCooldownMaxSeconds)
		}
		return model.CircuitStateOpen, in.now + nextCooldown
	default: // closed 或空
		if shouldTripCircuit(in) {
			return model.CircuitStateOpen, in.now + cooldown
		}
		// 仍 closed，但若上次有残留 unhealthy_until 已到期，清零。
		if in.prevUnhealthyTill > 0 && in.now >= in.prevUnhealthyTill {
			return model.CircuitStateClosed, 0
		}
		return model.CircuitStateClosed, in.prevUnhealthyTill
	}
}

// shouldTripCircuit 判断从 closed 是否应直接转 open：样本数足且成功率低，或鉴权类立刻熔断。
func shouldTripCircuit(in circuitInputs) bool {
	if !in.success && in.category == model.ErrorCategoryAuth {
		return true
	}
	if in.sampleCount < channelHealthMinRequestsForCircuit {
		return false
	}
	failureRate := 1.0 - in.successRate
	return failureRate >= channelHealthFailureRateThreshold
}

// classifyError 把一次失败归类到 ErrorCategory 枚举；优先级：TokenFactoryError > HTTPStatus > RawErr 内容。
// 命中不到时返回 ErrorCategoryOther；成功路径返回 ErrorCategoryNone。
func classifyError(in HealthSignalInput) string {
	if in.Success {
		return model.ErrorCategoryNone
	}
	status := in.HTTPStatus
	if in.TokenFactoryError != nil && in.TokenFactoryError.StatusCode > 0 {
		status = in.TokenFactoryError.StatusCode
	}
	switch {
	case status == http.StatusRequestTimeout || status == http.StatusGatewayTimeout:
		return model.ErrorCategoryTimeout
	case status == http.StatusTooManyRequests:
		return model.ErrorCategoryRateLimit
	case status == http.StatusUnauthorized || status == http.StatusForbidden, status == http.StatusPaymentRequired:
		return model.ErrorCategoryAuth
	case status >= 400 && status < 500:
		return model.ErrorCategoryBadRequest
	case status >= 500 && status < 600:
		return model.ErrorCategoryUpstream5xx
	}
	if in.RawErr != nil {
		msg := strings.ToLower(in.RawErr.Error())
		if errors.Is(in.RawErr, context.DeadlineExceeded) || strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline") {
			return model.ErrorCategoryTimeout
		}
		if strings.Contains(msg, "unmarshal") || strings.Contains(msg, "parse") {
			return model.ErrorCategoryParse
		}
	}
	return model.ErrorCategoryOther
}

// IsChannelHealthCircuitOpen 是 PR3 路由器的快速判据：true 表示该 (channel,model) 当前处于熔断窗口内。
// 仅看 unhealthy_until 与状态字段，不读其它指标，O(0) 开销（行数据已在 ModelTestResult 上下文）。
func IsChannelHealthCircuitOpen(row *model.ModelTestResult) bool {
	if row == nil {
		return false
	}
	if row.CircuitState == model.CircuitStateOpen && row.UnhealthyUntil > common.GetTimestamp() {
		return true
	}
	return false
}

// channelHealthEnvOverrideOrInt 预留：未来 PR2 把阈值改成按 routing_policies 读，过渡期可用环境变量临时调参。
// 例如 CHANNEL_HEALTH_FAILURE_RATE=0.6。当前未启用，函数留作 hook，避免后续频繁改动签名。
func channelHealthEnvOverrideOrInt(envKey string, def int) int { //nolint:unused
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func ptrInt(v int) *int             { return &v }
func ptrInt64(v int64) *int64       { return &v }
func ptrFloat(v float64) *float64   { return &v }
func ptrString(v string) *string    { return &v }
