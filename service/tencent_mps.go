package service

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	mps "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/mps/v20190612"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

// parsedCOSOutput 解析自全局 OutputPath，供 MPS 写出 COS 对象。
type parsedCOSOutput struct {
	Bucket    string
	Region    string
	OutputDir string // 必须以 / 开头和结尾
	BaseURL   string // https://bucket.cos.region.myqcloud.com
}

func strPtr(s string) *string { return &s }

func uint64Ptr(v uint64) *uint64 { return &v }

// parseVideoUpscaleOutputPath 解析 https://<bucket>.cos.<region>.myqcloud.com/<prefix>/
func parseVideoUpscaleOutputPath(raw string) (*parsedCOSOutput, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("output_path 为空")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("output_path 格式无效")
	}
	host := strings.ToLower(u.Hostname())
	const suffix = ".myqcloud.com"
	if !strings.HasSuffix(host, suffix) {
		return nil, fmt.Errorf("output_path 须为腾讯云 COS 域名")
	}
	// bucket.cos.ap-guangzhou.myqcloud.com
	rest := strings.TrimSuffix(host, suffix)
	parts := strings.Split(rest, ".cos.")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("output_path 无法解析 bucket/region")
	}
	dir := u.Path
	if dir == "" {
		dir = "/"
	}
	if !strings.HasPrefix(dir, "/") {
		dir = "/" + dir
	}
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return &parsedCOSOutput{
		Bucket:    parts[0],
		Region:    parts[1],
		OutputDir: dir,
		BaseURL:   fmt.Sprintf("https://%s.cos.%s.myqcloud.com", parts[0], parts[1]),
	}, nil
}

func newMpsClient(secretId, secretKey, region string) (*mps.Client, error) {
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.ReqMethod = "POST"
	cred := common.NewCredential(secretId, secretKey)
	return mps.NewClient(cred, region, cpf)
}

// SubmitVideoUpscaleTask 向腾讯云 MPS 提交超分转码任务，返回 MPS TaskId。
// 输入为公网可访问的视频 URL；输出写入全局配置的 COS 路径。
func SubmitVideoUpscaleTask(inputURL string, templateId uint64) (string, error) {
	setting := operation_setting.GetVideoUpscaleSetting()
	if setting == nil || !operation_setting.IsVideoUpscaleReady() {
		return "", fmt.Errorf("视频超分全局配置不完整")
	}
	inputURL = strings.TrimSpace(inputURL)
	if inputURL == "" || strings.HasPrefix(inputURL, "data:") {
		return "", fmt.Errorf("超分输入视频 URL 无效")
	}
	if templateId == 0 {
		return "", fmt.Errorf("超分模版 ID 为空")
	}
	out, err := parseVideoUpscaleOutputPath(setting.OutputPath)
	if err != nil {
		return "", err
	}
	client, err := newMpsClient(setting.SecretId, setting.SecretKey, out.Region)
	if err != nil {
		return "", fmt.Errorf("创建 MPS 客户端失败: %w", err)
	}
	req := mps.NewProcessMediaRequest()
	req.InputInfo = &mps.MediaInputInfo{
		Type: strPtr("URL"),
		UrlInputInfo: &mps.UrlInputInfo{
			Url: strPtr(inputURL),
		},
	}
	req.OutputStorage = &mps.TaskOutputStorage{
		Type: strPtr("COS"),
		CosOutputStorage: &mps.CosOutputStorage{
			Bucket: strPtr(out.Bucket),
			Region: strPtr(out.Region),
		},
	}
	req.OutputDir = strPtr(out.OutputDir)
	req.MediaProcessTask = &mps.MediaProcessTaskInput{
		TranscodeTaskSet: []*mps.TranscodeTaskInput{
			{Definition: uint64Ptr(templateId)},
		},
	}
	resp, err := client.ProcessMedia(req)
	if err != nil {
		return "", fmt.Errorf("提交 MPS 超分任务失败: %w", err)
	}
	if resp == nil || resp.Response == nil || resp.Response.TaskId == nil || *resp.Response.TaskId == "" {
		return "", fmt.Errorf("MPS 未返回任务 ID")
	}
	return *resp.Response.TaskId, nil
}

// VideoUpscaleMpsResult MPS 超分轮询结果。
type VideoUpscaleMpsResult struct {
	Finished   bool    // 任务是否结束（成功或失败）
	Success    bool    // 超分是否成功
	OutputURL  string  // 超分后视频公网 URL
	DurationSec float64 // 超分后视频时长（秒）
	FailReason string  // 失败原因
}

// DescribeVideoUpscaleTask 查询 MPS 超分任务状态。
func DescribeVideoUpscaleTask(mpsTaskId string) (*VideoUpscaleMpsResult, error) {
	mpsTaskId = strings.TrimSpace(mpsTaskId)
	if mpsTaskId == "" {
		return nil, fmt.Errorf("MPS 任务 ID 为空")
	}
	setting := operation_setting.GetVideoUpscaleSetting()
	if setting == nil || !operation_setting.IsVideoUpscaleReady() {
		return nil, fmt.Errorf("视频超分全局配置不完整")
	}
	out, err := parseVideoUpscaleOutputPath(setting.OutputPath)
	if err != nil {
		return nil, err
	}
	client, err := newMpsClient(setting.SecretId, setting.SecretKey, out.Region)
	if err != nil {
		return nil, fmt.Errorf("创建 MPS 客户端失败: %w", err)
	}
	req := mps.NewDescribeTaskDetailRequest()
	req.TaskId = strPtr(mpsTaskId)
	resp, err := client.DescribeTaskDetail(req)
	if err != nil {
		return nil, fmt.Errorf("查询 MPS 任务失败: %w", err)
	}
	if resp == nil || resp.Response == nil {
		return nil, fmt.Errorf("MPS 查询响应为空")
	}
	status := ""
	if resp.Response.Status != nil {
		status = strings.ToUpper(strings.TrimSpace(*resp.Response.Status))
	}
	result := &VideoUpscaleMpsResult{}
	if status == "" || status == "WAITING" || status == "PROCESSING" {
		return result, nil
	}
	if status != "FINISH" {
		result.Finished = true
		result.FailReason = fmt.Sprintf("MPS 未知状态: %s", status)
		return result, nil
	}
	result.Finished = true
	wf := resp.Response.WorkflowTask
	if wf == nil {
		result.FailReason = "MPS 完成但无工作流结果"
		return result, nil
	}
	if wf.ErrCode != nil && *wf.ErrCode != 0 {
		msg := ""
		if wf.Message != nil {
			msg = *wf.Message
		}
		result.FailReason = fmt.Sprintf("MPS 工作流失败: %s", msg)
		return result, nil
	}
	for _, item := range wf.MediaProcessResultSet {
		if item == nil || item.TranscodeTask == nil {
			continue
		}
		tc := item.TranscodeTask
		tcStatus := ""
		if tc.Status != nil {
			tcStatus = strings.ToUpper(strings.TrimSpace(*tc.Status))
		}
		if tcStatus == "FAIL" || tcStatus == "FAILED" {
			msg := ""
			if tc.Message != nil {
				msg = *tc.Message
			}
			result.FailReason = fmt.Sprintf("MPS 转码失败: %s", msg)
			return result, nil
		}
		if tcStatus != "SUCCESS" || tc.Output == nil {
			continue
		}
		if tc.Output.Duration != nil {
			result.DurationSec = *tc.Output.Duration
		}
		objectPath := ""
		if tc.Output.Path != nil {
			objectPath = strings.TrimSpace(*tc.Output.Path)
		}
		if objectPath != "" {
			// 实际 COS 对象路径优先：BaseURL + Output.Path（如 /video/output.mp4）。
			result.OutputURL = buildVideoUpscalePublicURL(setting.OutputPath, objectPath)
			result.Success = true
			return result, nil
		}
	}
	if wf.MetaData != nil && wf.MetaData.Duration != nil && result.DurationSec <= 0 {
		result.DurationSec = *wf.MetaData.Duration
	}
	if !result.Success {
		result.FailReason = "MPS 完成但未解析到超分输出"
	}
	return result, nil
}

// buildVideoUpscalePublicURL 构造超分后的公网视频地址。
// 优先：OutputPath 解析出的桶域名 + MPS Output.Path（完整对象路径，如 /video/output.mp4）；
// 回退：配置的 OutputPath 目录 + 文件名。
func buildVideoUpscalePublicURL(configuredOutputPath, mpsObjectPath string) string {
	configuredOutputPath = strings.TrimSpace(configuredOutputPath)
	mpsObjectPath = strings.TrimSpace(mpsObjectPath)
	if mpsObjectPath == "" {
		return strings.TrimRight(configuredOutputPath, "/") + "/"
	}
	if strings.HasPrefix(mpsObjectPath, "http://") || strings.HasPrefix(mpsObjectPath, "https://") {
		return mpsObjectPath
	}
	if out, err := parseVideoUpscaleOutputPath(configuredOutputPath); err == nil && out != nil {
		normalized := mpsObjectPath
		if !strings.HasPrefix(normalized, "/") {
			normalized = "/" + normalized
		}
		return joinCOSObjectURL(out.BaseURL, normalized)
	}
	name := path.Base(strings.ReplaceAll(mpsObjectPath, "\\", "/"))
	if name == "" || name == "." || name == "/" {
		return strings.TrimRight(configuredOutputPath, "/") + "/" + strings.TrimPrefix(mpsObjectPath, "/")
	}
	return strings.TrimRight(configuredOutputPath, "/") + "/" + name
}

func joinCOSObjectURL(baseURL, objectPath string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if objectPath == "" {
		return baseURL
	}
	if strings.HasPrefix(objectPath, "http://") || strings.HasPrefix(objectPath, "https://") {
		return objectPath
	}
	if !strings.HasPrefix(objectPath, "/") {
		objectPath = "/" + objectPath
	}
	return baseURL + objectPath
}
