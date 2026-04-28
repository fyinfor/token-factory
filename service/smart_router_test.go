package service

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/model"
)

// floatClose 在浮点比较时用，避免 1e-12 级误差让测试 flaky。
func floatClose(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// TestCandidatePerfFromMetrics_NoMetricsFallback 验证：当 (channel × model) 没有
// 健康行（mtr=nil，PR1 之前的旧渠道 / 新加渠道）时，函数完全退化到 ch.ResponseTime，
// 与 PR4 之前的行为一致；ThroughputTps 用 1/latSec 兜底；Healthy 必为 true。
func TestCandidatePerfFromMetrics_NoMetricsFallback(t *testing.T) {
	ch := &model.Channel{Id: 1, ResponseTime: 200}
	latSec, tps, healthy := candidatePerfFromMetrics(ch, nil, 1700_000_000)
	if !floatClose(latSec, 0.2) {
		t.Fatalf("latSec want 0.2, got %v", latSec)
	}
	if !floatClose(tps, 1.0/0.2) {
		t.Fatalf("tps want %v, got %v", 1.0/0.2, tps)
	}
	if !healthy {
		t.Fatalf("healthy want true when no mtr")
	}
}

// TestCandidatePerfFromMetrics_NoMetrics_ZeroResponseTime 验证：渠道粗粒度也是 0
// （刚建好还没测过）时，使用 0.001s 兜底，避免 latSec=0 让 tps=+Inf。
func TestCandidatePerfFromMetrics_NoMetrics_ZeroResponseTime(t *testing.T) {
	ch := &model.Channel{Id: 1, ResponseTime: 0}
	latSec, tps, healthy := candidatePerfFromMetrics(ch, nil, 1700_000_000)
	if !floatClose(latSec, 0.001) {
		t.Fatalf("latSec want 0.001 fallback, got %v", latSec)
	}
	if !floatClose(tps, 1000.0) {
		t.Fatalf("tps want 1000 (=1/0.001), got %v", tps)
	}
	if !healthy {
		t.Fatalf("healthy want true")
	}
}

// TestCandidatePerfFromMetrics_PrefersEwmaThenLastResponseTime 验证字段优先级：
//   - LatencyEwmaMs > 0 时优先用 EWMA（抹平单次抖动）；
//   - LatencyEwmaMs=0 且 LastResponseTime>0 时回退到 LastResponseTime；
//   - 两者都为 0 时退化到 ch.ResponseTime。
func TestCandidatePerfFromMetrics_PrefersEwmaThenLastResponseTime(t *testing.T) {
	ch := &model.Channel{Id: 1, ResponseTime: 1000}
	t.Run("ewma wins", func(t *testing.T) {
		mtr := &model.ModelTestResult{
			LatencyEwmaMs:    150, // EWMA = 0.15s
			LastResponseTime: 800, // 不该被采用
		}
		latSec, _, _ := candidatePerfFromMetrics(ch, mtr, 1700_000_000)
		if !floatClose(latSec, 0.15) {
			t.Fatalf("want 0.15s from EWMA, got %v", latSec)
		}
	})
	t.Run("last response time as fallback", func(t *testing.T) {
		mtr := &model.ModelTestResult{
			LatencyEwmaMs:    0,
			LastResponseTime: 800, // 0.8s
		}
		latSec, _, _ := candidatePerfFromMetrics(ch, mtr, 1700_000_000)
		if !floatClose(latSec, 0.8) {
			t.Fatalf("want 0.8s from LastResponseTime, got %v", latSec)
		}
	})
	t.Run("both zero falls back to channel", func(t *testing.T) {
		mtr := &model.ModelTestResult{LatencyEwmaMs: 0, LastResponseTime: 0}
		latSec, _, _ := candidatePerfFromMetrics(ch, mtr, 1700_000_000)
		if !floatClose(latSec, 1.0) {
			t.Fatalf("want 1.0s from ch.ResponseTime, got %v", latSec)
		}
	})
}

// TestCandidatePerfFromMetrics_ThroughputUsesActual 验证：mtr.LastThroughputTps>0
// 时直接用实测 TPS（不做 1/latSec 兜底），这是 sort=throughput 真正生效的关键。
// LastThroughputTps=0 时仍走 1/latSec 兜底，保留旧渠道参与排序。
func TestCandidatePerfFromMetrics_ThroughputUsesActual(t *testing.T) {
	ch := &model.Channel{Id: 1, ResponseTime: 500} // 1/0.5 = 2 tps
	t.Run("real tps wins", func(t *testing.T) {
		mtr := &model.ModelTestResult{LastThroughputTps: 42}
		_, tps, _ := candidatePerfFromMetrics(ch, mtr, 1700_000_000)
		if !floatClose(tps, 42) {
			t.Fatalf("want real tps=42, got %v", tps)
		}
	})
	t.Run("zero tps falls back to 1/latSec", func(t *testing.T) {
		mtr := &model.ModelTestResult{LastThroughputTps: 0}
		_, tps, _ := candidatePerfFromMetrics(ch, mtr, 1700_000_000)
		if !floatClose(tps, 2) {
			t.Fatalf("want 1/0.5=2 fallback, got %v", tps)
		}
	})
}

// TestCandidatePerfFromMetrics_OpenCircuitMarksUnhealthy 验证熔断三态行为：
//   - CircuitState=open 且 UnhealthyUntil > now → Healthy=false（router-engine 降级）；
//   - CircuitState=open 但已过期 → 视为已恢复，Healthy=true；
//   - CircuitState=closed 或 half_open → 不影响 Healthy。
func TestCandidatePerfFromMetrics_OpenCircuitMarksUnhealthy(t *testing.T) {
	ch := &model.Channel{Id: 1, ResponseTime: 200}
	const now int64 = 1700_000_000
	t.Run("open and not expired", func(t *testing.T) {
		mtr := &model.ModelTestResult{
			CircuitState:   model.CircuitStateOpen,
			UnhealthyUntil: now + 60,
		}
		_, _, healthy := candidatePerfFromMetrics(ch, mtr, now)
		if healthy {
			t.Fatalf("want unhealthy while circuit open")
		}
	})
	t.Run("open but expired", func(t *testing.T) {
		mtr := &model.ModelTestResult{
			CircuitState:   model.CircuitStateOpen,
			UnhealthyUntil: now - 1,
		}
		_, _, healthy := candidatePerfFromMetrics(ch, mtr, now)
		if !healthy {
			t.Fatalf("want healthy when UnhealthyUntil expired")
		}
	})
	t.Run("closed", func(t *testing.T) {
		mtr := &model.ModelTestResult{CircuitState: model.CircuitStateClosed}
		_, _, healthy := candidatePerfFromMetrics(ch, mtr, now)
		if !healthy {
			t.Fatalf("want healthy when closed")
		}
	})
	t.Run("half_open", func(t *testing.T) {
		mtr := &model.ModelTestResult{
			CircuitState:   model.CircuitStateHalfOpen,
			UnhealthyUntil: now + 60, // 即使 UntilUnhealthy 还没到，half_open 不应被视为开路
		}
		_, _, healthy := candidatePerfFromMetrics(ch, mtr, now)
		if !healthy {
			t.Fatalf("want healthy when half_open (probe phase)")
		}
	})
}
