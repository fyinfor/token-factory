package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
)

// ────────────────────────────────────────────────────────────────────────────
// 官方预设缓存（TTL 15 分钟，避免每次 auto_meta 都重复抓取）
// ────────────────────────────────────────────────────────────────────────────

const (
	officialModelsPresetURL = "https://basellm.github.io/llm-metadata/api/newapi/models.json"
	presetCacheTTL          = 15 * time.Minute
)

type officialModelEntry struct {
	ModelName string          `json:"model_name"`
	Endpoints json.RawMessage `json:"endpoints"`
	Tags      string          `json:"tags"`
	VendorName string         `json:"vendor_name"`
	Description string        `json:"description"`
	Icon      string          `json:"icon"`
	NameRule  int             `json:"name_rule"`
	Status    int             `json:"status"`
}

type officialPresetEnvelope struct {
	Success bool                 `json:"success"`
	Data    []officialModelEntry `json:"data"`
}

var (
	presetMu      sync.RWMutex
	presetByName  map[string]officialModelEntry
	presetFetchAt time.Time
)

// fetchOfficialPreset 获取官方模型预设（带本地缓存）。
// 缓存未过期时直接返回内存副本；过期或首次调用时请求远端。
func fetchOfficialPreset(ctx context.Context) map[string]officialModelEntry {
	presetMu.RLock()
	if presetByName != nil && time.Since(presetFetchAt) < presetCacheTTL {
		m := presetByName
		presetMu.RUnlock()
		return m
	}
	presetMu.RUnlock()

	// 升级为写锁后二次检查，防止并发重复抓取
	presetMu.Lock()
	defer presetMu.Unlock()
	if presetByName != nil && time.Since(presetFetchAt) < presetCacheTTL {
		return presetByName
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, officialModelsPresetURL, nil)
	if err != nil {
		return presetByName // 失败时沿用旧缓存
	}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return presetByName
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return presetByName
	}

	// 兼容两种格式：envelope{ success, data:[] } 或 直接 []
	var env officialPresetEnvelope
	if err := json.Unmarshal(body, &env); err == nil && len(env.Data) > 0 {
		m := make(map[string]officialModelEntry, len(env.Data))
		for _, e := range env.Data {
			if e.ModelName != "" {
				m[e.ModelName] = e
			}
		}
		presetByName = m
		presetFetchAt = time.Now()
		return presetByName
	}
	// 尝试直接解析为数组
	var arr []officialModelEntry
	if err := json.Unmarshal(body, &arr); err == nil {
		m := make(map[string]officialModelEntry, len(arr))
		for _, e := range arr {
			if e.ModelName != "" {
				m[e.ModelName] = e
			}
		}
		presetByName = m
		presetFetchAt = time.Now()
	}
	return presetByName
}

// ────────────────────────────────────────────────────────────────────────────
// 模型名称规则推断
// ────────────────────────────────────────────────────────────────────────────

// inferEndpoints 根据模型名推断 Endpoints JSON 字符串（如 `["openai"]`）。
// 推断顺序：Embedding → Rerank → Image → Video → Chat（默认）
func inferEndpoints(name string) string {
	lower := strings.ToLower(name)

	switch {
	// Embedding
	case strings.Contains(lower, "embed"),
		strings.HasPrefix(lower, "bge-"),
		strings.HasPrefix(lower, "m3e-"),
		strings.Contains(lower, "jina-embed"):
		return `["embeddings"]`

	// Rerank
	case strings.Contains(lower, "rerank"),
		strings.Contains(lower, "jina-rerank"):
		return `["jina-rerank"]`

	// Image generation
	case strings.Contains(lower, "dall-e"),
		strings.Contains(lower, "sdxl"),
		strings.Contains(lower, "stable-diffusion"),
		strings.Contains(lower, "wanx"),
		strings.Contains(lower, "kolors"),
		strings.Contains(lower, "cogview"),
		strings.Contains(lower, "hunyuan-dit"),
		strings.Contains(lower, "flux"),
		matchesPattern(lower, []string{"image-alpha", "imagen-", "text-to-image"}):
		return `["image-generation"]`

	// Video generation
	case strings.Contains(lower, "video-generation"),
		strings.Contains(lower, "kling"),
		strings.Contains(lower, "vidu"),
		matchesPattern(lower, []string{"video-01", "video-02"}):
		return `["openai-video"]`

	// 默认：Chat (openai-compatible)
	default:
		return `["openai"]`
	}
}

// inferTags 根据模型名推断标签（逗号分隔字符串）。
func inferTags(name string) string {
	lower := strings.ToLower(name)
	var tags []string
	seen := make(map[string]bool)

	add := func(t string) {
		if !seen[t] {
			seen[t] = true
			tags = append(tags, t)
		}
	}

	// 视觉/多模态
	if strings.Contains(lower, "vision") ||
		strings.Contains(lower, "-vl") ||
		strings.Contains(lower, "omni") ||
		strings.Contains(lower, "visual") {
		add("vision")
	}
	// 推理增强
	if strings.Contains(lower, "thinking") ||
		strings.Contains(lower, "reasoner") ||
		strings.Contains(lower, "-r1") ||
		strings.Contains(lower, "-r2") ||
		strings.HasPrefix(lower, "o1") ||
		strings.HasPrefix(lower, "o3") ||
		strings.Contains(lower, "-think") ||
		strings.Contains(lower, "qwq") {
		add("reasoning")
	}
	// 代码
	if strings.Contains(lower, "code") ||
		strings.Contains(lower, "coder") ||
		strings.Contains(lower, "codex") ||
		strings.Contains(lower, "codestral") ||
		strings.Contains(lower, "deepseek-coder") {
		add("coding")
	}
	// Embedding
	if strings.Contains(lower, "embed") ||
		strings.HasPrefix(lower, "bge-") ||
		strings.HasPrefix(lower, "m3e-") {
		add("embedding")
	}
	// Rerank
	if strings.Contains(lower, "rerank") {
		add("rerank")
	}
	// Image
	if strings.Contains(lower, "dall-e") ||
		strings.Contains(lower, "sdxl") ||
		strings.Contains(lower, "flux") ||
		strings.Contains(lower, "image-generation") {
		add("image")
	}
	// 音频
	if strings.Contains(lower, "whisper") ||
		strings.Contains(lower, "-asr") ||
		strings.Contains(lower, "tts") {
		add("audio")
	}
	// 轻量/经济型
	if strings.Contains(lower, "mini") ||
		strings.Contains(lower, "lite") ||
		strings.Contains(lower, "tiny") ||
		strings.Contains(lower, "nano") ||
		strings.Contains(lower, "small") ||
		strings.Contains(lower, "flash") ||
		strings.Contains(lower, "haiku") {
		add("budget")
	}

	return strings.Join(tags, ",")
}

// ────────────────────────────────────────────────────────────────────────────
// 标签过滤：移除不适合用户分类使用的标签
// ────────────────────────────────────────────────────────────────────────────

// validTagSet 定义允许作为模型分类标签的合法标签集合（小写）。
// 不在此集合中的标签将被过滤掉（如上下文窗口大小 "262.1K"、"128K" 等数值型标签）。
var validTagSet = map[string]bool{
	// 能力分类
	"reasoning":  true,
	"tools":      true,
	"files":      true,
	"vision":     true,
	"coding":     true,
	"code":       true,
	"embedding":  true,
	"rerank":     true,
	"image":      true,
	"audio":      true,
	"video":      true,
	"budget":     true,
	// 模型属性
	"open weights": true,
	"open source":  true,
	"proprietary":  true,
	"local":        true,
	"cloud":        true,
	"multilingual": true,
	// 通用分类
	"chat":       true,
	"completion": true,
	"instruct":   true,
	"base":       true,
	"fine-tuned": true,
	"lora":       true,
}

// filterTags 过滤逗号分隔的标签字符串，只保留合法的分类标签。
// 用于清理官方预设中可能包含的上下文窗口大小（如 "262.1K"、"128K"）等
// 不适合作为用户筛选分类的数值型标签。
func filterTags(tagsStr string) string {
	if tagsStr == "" {
		return ""
	}
	parts := strings.Split(tagsStr, ",")
	var filtered []string
	for _, p := range parts {
		tag := strings.TrimSpace(p)
		if tag == "" {
			continue
		}
		// 精确匹配合法标签（不区分大小写）
		if validTagSet[strings.ToLower(tag)] {
			filtered = append(filtered, tag)
		}
	}
	return strings.Join(filtered, ",")
}

// matchesPattern 检查 lower 是否包含 patterns 中的任意一个。
func matchesPattern(lower string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// ────────────────────────────────────────────────────────────────────────────
// 供应商推断（VendorID）
// ────────────────────────────────────────────────────────────────────────────

// vendorKeywordAliases 将"模型名关键词"映射到"供应商名关键词"（小写）。
// 匹配策略：先在模型名中搜索 key，命中后再用 values 匹配数据库中 Vendor.Name（小写子串）。
var vendorKeywordAliases = []struct {
	modelKW   string   // 在模型名中搜索（小写）
	vendorKWs []string // 在 Vendor.Name 中任意一个命中即可（小写）
}{
	{"claude", []string{"anthropic"}},
	{"gemini", []string{"google"}},
	{"gpt", []string{"openai"}},
	{"dall-e", []string{"openai"}},
	{"whisper", []string{"openai"}},
	{"o1-", []string{"openai"}},
	{"o3-", []string{"openai"}},
	{"o4-", []string{"openai"}},
	{"llama", []string{"meta"}},
	{"mistral", []string{"mistral"}},
	{"mixtral", []string{"mistral"}},
	{"codestral", []string{"mistral"}},
	{"deepseek", []string{"deepseek"}},
	{"qwen", []string{"alibaba", "qwen", "tongyi", "aliyun"}},
	{"moonshot", []string{"moonshot"}},
	{"kimi", []string{"moonshot"}},
	{"doubao", []string{"bytedance", "volcengine", "volcano"}},
	{"ernie", []string{"baidu"}},
	{"wenxin", []string{"baidu"}},
	{"hunyuan", []string{"tencent"}},
	{"spark", []string{"xunfei", "iflytek"}},
	{"glm", []string{"zhipu", "chatglm"}},
	{"chatglm", []string{"zhipu"}},
	{"yi-", []string{"lingyiwanwu", "01ai", "zero-one"}},
	{"minimax", []string{"minimax"}},
	{"abab", []string{"minimax"}},
	{"flux", []string{"black forest", "blackforest"}},
	{"stable-diffusion", []string{"stability"}},
	{"sdxl", []string{"stability"}},
	{"cohere", []string{"cohere"}},
	{"command-r", []string{"cohere"}},
	{"perplexity", []string{"perplexity"}},
	{"jina", []string{"jina"}},
	{"suno", []string{"suno"}},
	{"kling", []string{"kling", "kuaishou"}},
	{"vidu", []string{"vidu", "shengshu"}},
	{"cogview", []string{"zhipu"}},
	{"internlm", []string{"shanghaiai", "intern"}},
	{"baichuan", []string{"baichuan"}},
	{"xai", []string{"xai"}},
	{"grok", []string{"xai"}},
}

// buildVendorIndex 一次性从 DB 中加载所有 Vendor，构建 name.lower → id 的映射。
func buildVendorIndex() map[string]int {
	vendors, err := model.GetAllVendors(0, 2000)
	if err != nil || len(vendors) == 0 {
		return nil
	}
	idx := make(map[string]int, len(vendors))
	for _, v := range vendors {
		idx[strings.ToLower(v.Name)] = v.Id
	}
	return idx
}

// inferVendorID 根据模型名在 vendorIdx 中查找最可能的供应商 ID，找不到返回 0。
func inferVendorID(modelName string, vendorIdx map[string]int) int {
	if len(vendorIdx) == 0 {
		return 0
	}
	lower := strings.ToLower(modelName)

	for _, rule := range vendorKeywordAliases {
		if !strings.Contains(lower, rule.modelKW) {
			continue
		}
		// 模型名匹配到关键词 → 在 vendorIdx 中搜索供应商名关键词
		for vendorNameLower, id := range vendorIdx {
			for _, vkw := range rule.vendorKWs {
				if strings.Contains(vendorNameLower, vkw) {
					return id
				}
			}
		}
	}

	// 兜底：尝试用 vendorIdx 中的供应商名直接匹配模型名（如模型名直接含供应商名）
	for vendorNameLower, id := range vendorIdx {
		if len(vendorNameLower) >= 4 && strings.Contains(lower, vendorNameLower) {
			return id
		}
	}
	return 0
}

// ────────────────────────────────────────────────────────────────────────────
// 对外接口：AutoCreateMissingModelMeta
// ────────────────────────────────────────────────────────────────────────────

// AutoMetaItem 单个模型的自动推断结果。
type AutoMetaItem struct {
	ModelName string `json:"model_name"`
	// "official"：来自官方预设；"inferred"：名称规则推断；"exists"：已有记录跳过
	Source    string `json:"source"`
	Endpoints string `json:"endpoints"`
	Tags      string `json:"tags"`
	VendorID  int    `json:"vendor_id,omitempty"`
	Err       string `json:"err,omitempty"`
}

// AutoCreateMissingModelMeta 对给定模型名列表，为缺少 model_meta 记录的模型
// 自动推断并创建元数据（先查官方预设，再用名称规则兜底）。
// 返回每个模型的处理结果。
func AutoCreateMissingModelMeta(ctx context.Context, modelNames []string) []AutoMetaItem {
	if len(modelNames) == 0 {
		return nil
	}

	// 1. 找出已存在的模型名（跳过）
	existingNames, _ := model.GetExistingModelNames(modelNames)
	existingSet := make(map[string]bool, len(existingNames))
	for _, n := range existingNames {
		existingSet[n] = true
	}

	// 2. 拉取官方预设（带缓存）
	preset := fetchOfficialPreset(ctx)

	// 3. 构建供应商索引（vendor name lower → id），用于 VendorID 推断
	vendorIdx := buildVendorIndex()

	results := make([]AutoMetaItem, 0, len(modelNames))

	for _, name := range modelNames {
		// 已存在：跳过
		if existingSet[name] {
			results = append(results, AutoMetaItem{
				ModelName: name,
				Source:    "exists",
			})
			continue
		}

		item := AutoMetaItem{ModelName: name}

		// 3a. 优先：官方预设精确匹配
		if entry, ok := preset[name]; ok {
			item.Source = "official"
			if len(entry.Endpoints) > 0 && string(entry.Endpoints) != "null" {
				item.Endpoints = string(entry.Endpoints)
			} else {
				item.Endpoints = inferEndpoints(name)
			}
			item.Tags = filterTags(entry.Tags)
			if item.Tags == "" {
				item.Tags = inferTags(name)
			}

			vendorID := inferVendorID(name, vendorIdx)
			item.VendorID = vendorID
			mi := &model.Model{
				ModelName:    name,
				Description:  entry.Description,
				Icon:         entry.Icon,
				Tags:         item.Tags,
				Endpoints:    item.Endpoints,
				VendorID:     vendorID,
				Status:       chooseModelStatus(entry.Status),
				NameRule:     entry.NameRule,
				SyncOfficial: 1,
			}
			if err := mi.Insert(); err != nil {
				item.Err = fmt.Sprintf("DB error: %v", err)
			}
		} else {
			// 3b. 兜底：名称规则推断
			item.Source = "inferred"
			item.Endpoints = inferEndpoints(name)
			item.Tags = inferTags(name)

			vendorID := inferVendorID(name, vendorIdx)
			item.VendorID = vendorID
			mi := &model.Model{
				ModelName:    name,
				Tags:         item.Tags,
				Endpoints:    item.Endpoints,
				VendorID:     vendorID,
				Status:       1,
				SyncOfficial: 1,
			}
			if err := mi.Insert(); err != nil {
				item.Err = fmt.Sprintf("DB error: %v", err)
			}
		}

		results = append(results, item)
	}

	return results
}

func chooseModelStatus(upstreamStatus int) int {
	if upstreamStatus == 0 {
		return 1
	}
	return upstreamStatus
}
