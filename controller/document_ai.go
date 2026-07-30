package controller

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"

	"github.com/gin-gonic/gin"
)

const (
	maxDocumentAIPromptLength = 40000
	maxDocumentAILength       = 120000

	defaultDocumentPolishPrompt = `你是一名资深 API 文档技术编辑。请依据原始文档整理出可直接发布、便于开发者调用的 Markdown 文档。

核心原则：
- 原始文档是唯一事实来源。不得猜测或补充未明确提供的字段、默认值、状态、错误码、轮询间隔、超时或能力范围。
- 保留所有接口路径、请求字段、响应结构、限制和必要流程；合并重复说明和重复示例。
- 保持原文语言，直接输出最终 Markdown。不要输出思考过程、修改说明、检查清单或额外代码围栏。
- 保留并统一使用 {{base_url}}、{{model}}、{{api_key}}，不要改写第三方素材地址和业务回调地址。

文档结构按实际内容组织，不存在的能力不要强行增加章节：
1. 使用一个 H1 标题，并简要说明适用模型、BaseURL、鉴权和接口用途。
2. 提供“快速开始”，包含最小可运行请求、响应和取得最终结果所需的下一步。
3. 快速开始中的同一个请求可使用 :::code-group 给出 cURL、JavaScript、Python、Java 示例。不同语言必须保持相同 URL、字段和业务语义。
4. 仅当原文为异步接口时，增加“异步任务流程”，完整说明创建任务、取得任务 ID、查询状态、成功取值和失败处理。
5. 请求参数按业务含义分组，每个字段只完整解释一次。表格最多使用“字段 / 类型 / 必填 / 说明”四列。
6. 仅当原文包含不同输入组合或调用目的时，增加“常用场景”，先概括差异，再保留有实际差异的可复制示例。
7. 保留成功响应、失败响应、状态、错误与限制。状态较少时使用“状态值 / 含义 / 下一步”三列表格，下一步只写原文明确给出的动作。
8. 素材库、回调、尾帧等独有能力放在对应章节，并保留必要示例。

排版要求：
- 只允许一个 H1；主要章节使用 H2，子章节使用 H3；不要手写目录或机械编号。
- 删除重复请求头、重复参数表，以及请求结构完全相同且只替换提示词的示例，但不得删除独有字段组合和完整响应。
- 不强制压缩比例或固定示例数量，以事实完整、调用顺序清楚、示例可复制为准。
- 异步接口不能把一次请求机械扩写成虚假的自动轮询代码。
- 保证 Markdown 表格、代码围栏和 :::code-group 成对闭合。`

	defaultDocumentTranslatePrompt = `Translate the supplied Chinese API documentation into clear, publishable technical English.

Requirements:
- Output only the complete translated Markdown. Do not include reasoning, notes, or an outer code fence.
- Preserve the document structure and all factual content without adding, removing, or correcting API behavior.
- Translate prose, headings, table labels, and reader-facing comments.
- Preserve URLs, endpoint paths, HTTP methods, JSON keys, field names, identifiers, commands, code syntax, and the placeholders {{base_url}}, {{model}}, and {{api_key}} exactly.
- Preserve fenced-code language tags and :::code-group containers.
- Keep asynchronous task creation, status queries, success and failure handling, result retrieval, limitations, and examples complete.`
)

type documentAIPromptPayload struct {
	PolishPrompt    string `json:"polish_prompt"`
	TranslatePrompt string `json:"translate_prompt"`
}

type documentAIGenerateRequest struct {
	Model    string `json:"model"`
	Action   string `json:"action"`
	Document string `json:"document"`
}

func getEffectiveDocumentAIPrompts() (documentAIPromptPayload, int64, bool, error) {
	settings, err := model.GetDocumentAIPromptSettings()
	if err != nil {
		return documentAIPromptPayload{}, 0, false, err
	}
	if settings == nil {
		return documentAIPromptPayload{
			PolishPrompt:    defaultDocumentPolishPrompt,
			TranslatePrompt: defaultDocumentTranslatePrompt,
		}, 0, true, nil
	}
	return documentAIPromptPayload{
		PolishPrompt:    settings.PolishPrompt,
		TranslatePrompt: settings.TranslatePrompt,
	}, settings.UpdatedTime, false, nil
}

func writeDocumentAIPrompts(c *gin.Context) {
	prompts, updatedTime, isDefault, err := getEffectiveDocumentAIPrompts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"polish_prompt":    prompts.PolishPrompt,
			"translate_prompt": prompts.TranslatePrompt,
			"updated_time":     updatedTime,
			"is_default":       isDefault,
		},
	})
}

func GetDocumentAIPrompts(c *gin.Context) {
	writeDocumentAIPrompts(c)
}

func UpdateDocumentAIPrompts(c *gin.Context) {
	var payload documentAIPromptPayload
	if err := common.DecodeJson(c.Request.Body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的请求参数"})
		return
	}
	payload.PolishPrompt = strings.TrimSpace(payload.PolishPrompt)
	payload.TranslatePrompt = strings.TrimSpace(payload.TranslatePrompt)
	if payload.PolishPrompt == "" || payload.TranslatePrompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "润色和翻译提示词不能为空"})
		return
	}
	if utf8.RuneCountInString(payload.PolishPrompt) > maxDocumentAIPromptLength ||
		utf8.RuneCountInString(payload.TranslatePrompt) > maxDocumentAIPromptLength {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "单个提示词不能超过 40000 个字符"})
		return
	}
	if err := model.SaveDocumentAIPromptSettings(&model.DocumentAIPromptSettings{
		PolishPrompt:    payload.PolishPrompt,
		TranslatePrompt: payload.TranslatePrompt,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	writeDocumentAIPrompts(c)
}

func ResetDocumentAIPrompts(c *gin.Context) {
	if err := model.ResetDocumentAIPromptSettings(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	writeDocumentAIPrompts(c)
}

func PrepareDocumentAIRequest(c *gin.Context) {
	var payload documentAIGenerateRequest
	if err := common.UnmarshalBodyReusable(c, &payload); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的请求参数"})
		return
	}
	payload.Model = strings.TrimSpace(payload.Model)
	payload.Action = strings.ToLower(strings.TrimSpace(payload.Action))
	payload.Document = strings.TrimSpace(payload.Document)
	if payload.Model == "" || utf8.RuneCountInString(payload.Model) > 256 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "message": "请选择有效的文本模型"})
		return
	}
	if payload.Action != "polish" && payload.Action != "translate" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "message": "不支持的文档处理方式"})
		return
	}
	if payload.Document == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "message": "文档内容不能为空"})
		return
	}
	if utf8.RuneCountInString(payload.Document) > maxDocumentAILength {
		c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "文档不能超过 120000 个字符"})
		return
	}

	prompts, _, _, err := getEffectiveDocumentAIPrompts()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	systemPrompt := prompts.PolishPrompt
	temperature := 0.2
	userPrompt := "请依据系统规范整理下面的原始 API 文档。\n\n目标模型：{{model}}\n请以原文为唯一事实来源，输出可直接发布的完整 Markdown。\n\n--- 原始文档开始 ---\n\n" + payload.Document + "\n\n--- 原始文档结束 ---"
	if payload.Action == "translate" {
		systemPrompt = prompts.TranslatePrompt
		temperature = 0.1
		userPrompt = "Translate the following API documentation into English according to the system requirements.\n\nTarget model: {{model}}\n\n--- Source document begins ---\n\n" + payload.Document + "\n\n--- Source document ends ---"
	}

	relayRequest := dto.GeneralOpenAIRequest{
		Model: payload.Model,
		Messages: []dto.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Stream:      common.GetPointer(true),
		Temperature: common.GetPointer(temperature),
	}
	body, err := common.Marshal(relayRequest)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "message": "构建 AI 请求失败"})
		return
	}
	if err := common.ReplaceRequestBody(c, body); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "message": "构建 AI 请求失败"})
		return
	}

	// Reuse the Playground relay so channel selection, billing, retries, logs, and SSE stay consistent.
	c.Request.URL.Path = "/api/playground/chat/completions"
	c.Request.URL.RawPath = ""
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("relay_mode", relayconstant.RelayModeChatCompletions)
	c.Next()
}
