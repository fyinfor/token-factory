package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	cloudauth "github.com/alibabacloud-go/cloudauth-20190307/v4/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/dara"
	"gorm.io/gorm"
)

const realNameVerificationValidity = 15 * time.Minute

var realNameVerificationMessageMutex sync.Mutex

type RealNameVerificationStatus struct {
	Status        string  `json:"status"`
	VerifiedAt    *int64  `json:"verified_at,omitempty"`
	ExpiresAt     int64   `json:"expires_at"`
	RewardEnabled bool    `json:"reward_enabled"`
	RewardAmount  float64 `json:"reward_amount"`
	RewardGranted bool    `json:"reward_granted"`
}

func CreateRealNameVerification(userId int) (*model.RealNameVerification, error) {
	if !setting.AliyunRealNameVerificationConfigured() {
		return nil, errors.New("实名认证服务暂未配置")
	}
	return model.CreateRealNameVerification(userId, common.GetRandomString(48), time.Now().Add(realNameVerificationValidity))
}

func StartAliyunRealNameVerification(ctx context.Context, launchToken, metaInfo, certName, certNo string) (string, error) {
	record, err := model.GetRealNameVerificationByLaunchToken(strings.TrimSpace(launchToken))
	if err != nil || record.Status != model.RealNameVerificationStatusPending || time.Now().After(record.ExpiresAt) {
		return "", errors.New("认证链接无效或已失效")
	}
	if record.CertifyId != nil && *record.CertifyId != "" {
		return "", errors.New("认证已启动，请勿重复提交")
	}
	if strings.TrimSpace(metaInfo) == "" {
		return "", errors.New("无法获取当前设备信息，请使用支持的手机浏览器重试")
	}
	if strings.TrimSpace(certName) == "" || strings.TrimSpace(certNo) == "" {
		return "", errors.New("请填写真实姓名和身份证号码")
	}
	sceneID, err := strconv.ParseInt(strings.TrimSpace(setting.AliyunRealNameVerificationSceneID), 10, 64)
	if err != nil || sceneID <= 0 {
		return "", errors.New("阿里云实名认证场景 ID 未配置或格式无效")
	}
	client, err := newAliyunRealNameVerificationClient()
	if err != nil {
		return "", err
	}
	returnURL, err := buildAliyunRealNameVerificationURL(
		setting.AliyunRealNameVerificationReturnURL,
		fmt.Sprintf("%s/real-name/result", strings.TrimRight(system_setting.ServerAddress, "/")),
		launchToken,
	)
	if err != nil {
		return "", fmt.Errorf("实名认证跳转地址配置无效: %w", err)
	}
	request := (&cloudauth.InitFaceVerifyRequest{}).
		SetProductCode(setting.AliyunRealNameVerificationProductCode).
		SetSceneId(sceneID).
		SetModel(setting.AliyunRealNameVerificationModel).
		SetCertType("IDENTITY_CARD").
		SetCertName(certName).
		SetCertNo(certNo).
		SetCertifyUrlType("H5").
		SetMetaInfo(metaInfo).
		SetReturnUrl(returnURL).
		SetCallbackToken(launchToken).
		SetOuterOrderNo(common.GetRandomString(32))
	if strings.TrimSpace(setting.AliyunRealNameVerificationCallbackURL) != "" {
		callbackURL, callbackErr := buildAliyunRealNameVerificationURL(
			setting.AliyunRealNameVerificationCallbackURL,
			"",
			launchToken,
		)
		if callbackErr != nil {
			return "", fmt.Errorf("实名认证回调地址配置无效: %w", callbackErr)
		}
		request.SetCallbackUrl(callbackURL)
	}
	response, err := client.InitFaceVerifyWithContext(ctx, request, &dara.RuntimeOptions{})
	if err != nil {
		return "", fmt.Errorf("发起阿里云实名认证失败: %w", err)
	}
	if response == nil || response.Body == nil {
		return "", errors.New("阿里云实名认证未返回响应")
	}
	if response.Body.ResultObject == nil || response.Body.ResultObject.CertifyId == nil || response.Body.ResultObject.CertifyUrl == nil || strings.TrimSpace(*response.Body.ResultObject.CertifyUrl) == "" {
		code := stringValue(response.Body.Code)
		message := stringValue(response.Body.Message)
		diagnostic := fmt.Sprintf(
			"阿里云实名认证未返回认证链接（Code=%s，Message=%s，RequestId=%s）",
			code,
			message,
			stringValue(response.Body.RequestId),
		)
		common.SysLog(diagnostic)
		if isAliyunInvalidCertNoError(code, message) {
			return "", errors.New("身份证号错误")
		}
		return "", errors.New(diagnostic)
	}
	if err = model.DB.Model(record).Update("certify_id", *response.Body.ResultObject.CertifyId).Error; err != nil {
		return "", err
	}
	return *response.Body.ResultObject.CertifyUrl, nil
}

func GetAliyunRealNameVerificationStatus(ctx context.Context, launchToken string) (*RealNameVerificationStatus, error) {
	record, err := model.GetRealNameVerificationByLaunchToken(strings.TrimSpace(launchToken))
	if err != nil {
		return nil, errors.New("认证记录不存在")
	}
	if record.Status == model.RealNameVerificationStatusPending && time.Now().After(record.ExpiresAt) {
		if err = model.DB.Model(record).Update("status", model.RealNameVerificationStatusExpired).Error; err != nil {
			return nil, err
		}
		record.Status = model.RealNameVerificationStatusExpired
	}
	if record.Status == model.RealNameVerificationStatusPending && record.CertifyId != nil && *record.CertifyId != "" {
		if err = refreshAliyunRealNameVerificationStatus(ctx, record); err != nil {
			return nil, err
		}
	}
	if record.VerifiedAt != nil {
		if messageErr := publishRealNameVerificationSuccessMessage(record); messageErr != nil {
			common.SysLog("\u5b9e\u540d\u8ba4\u8bc1\u6210\u529f\u7ad9\u5185\u6d88\u606f\u8865\u53d1\u5931\u8d25: " + messageErr.Error())
		}
	}
	return toRealNameVerificationStatus(record), nil
}

func GetCurrentUserRealNameVerificationStatus(ctx context.Context, userId int) (*RealNameVerificationStatus, error) {
	if err := expireStaleRealNameVerificationRecords(userId); err != nil {
		return nil, err
	}
	record, err := model.GetLatestActiveRealNameVerificationByUserId(userId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &RealNameVerificationStatus{Status: "unverified", RewardEnabled: setting.AliyunRealNameVerificationRewardEnabled, RewardAmount: setting.AliyunRealNameVerificationRewardAmount}, nil
	}
	if err != nil {
		return nil, err
	}
	if record.Status == model.RealNameVerificationStatusPending && time.Now().After(record.ExpiresAt) {
		if err = model.DB.Model(record).Update("status", model.RealNameVerificationStatusExpired).Error; err != nil {
			return nil, err
		}
		record.Status = model.RealNameVerificationStatusExpired
	}
	if record.Status == model.RealNameVerificationStatusPending && record.CertifyId != nil && *record.CertifyId != "" {
		if err = refreshAliyunRealNameVerificationStatus(ctx, record); err != nil {
			return nil, err
		}
	}
	if record.VerifiedAt != nil {
		if messageErr := publishRealNameVerificationSuccessMessage(record); messageErr != nil {
			common.SysLog("\u5b9e\u540d\u8ba4\u8bc1\u6210\u529f\u7ad9\u5185\u6d88\u606f\u8865\u53d1\u5931\u8d25: " + messageErr.Error())
		}
	}
	return toRealNameVerificationStatus(record), nil
}

func expireStaleRealNameVerificationRecords(userId int) error {
	return model.DB.Model(&model.RealNameVerification{}).
		Where("user_id = ? AND status = ? AND expires_at < ?", userId, model.RealNameVerificationStatusPending, time.Now()).
		Update("status", model.RealNameVerificationStatusExpired).Error
}

func refreshAliyunRealNameVerificationStatus(ctx context.Context, record *model.RealNameVerification) error {
	client, err := newAliyunRealNameVerificationClient()
	if err != nil {
		return err
	}
	sceneID, err := strconv.ParseInt(strings.TrimSpace(setting.AliyunRealNameVerificationSceneID), 10, 64)
	if err != nil || sceneID <= 0 {
		return errors.New("阿里云实名认证场景 ID 未配置或格式无效")
	}
	request := (&cloudauth.DescribeFaceVerifyRequest{}).
		SetCertifyId(*record.CertifyId).
		SetSceneId(sceneID)
	response, err := client.DescribeFaceVerifyWithContext(ctx, request, &dara.RuntimeOptions{})
	if err != nil {
		return fmt.Errorf("查询阿里云实名认证结果失败: %w", err)
	}
	if response == nil || response.Body == nil || response.Body.ResultObject == nil {
		return errors.New("阿里云实名认证未返回认证结果")
	}
	if stringValue(response.Body.Code) != "200" {
		return fmt.Errorf(
			"阿里云实名认证结果查询失败（Code=%s，Message=%s，RequestId=%s）",
			stringValue(response.Body.Code),
			stringValue(response.Body.Message),
			stringValue(response.Body.RequestId),
		)
	}
	result := response.Body.ResultObject
	if result.Passed == nil {
		return nil
	}
	providerCode := stringValue(result.SubCode)
	switch *result.Passed {
	case "T":
		now := time.Now()
		rewardQuota := 0
		if setting.AliyunRealNameVerificationRewardEnabled && setting.AliyunRealNameVerificationRewardAmount > 0 {
			rewarded, rewardErr := model.HasReceivedRealNameVerificationReward(record.UserId)
			if rewardErr != nil {
				return rewardErr
			}
			if !rewarded {
				rewardQuota = int(math.Round(setting.AliyunRealNameVerificationRewardAmount * common.QuotaPerUnit))
			}
		}
		err = model.DB.Transaction(func(tx *gorm.DB) error {
			updates := map[string]any{"status": model.RealNameVerificationStatusPassed, "verified_at": now, "provider_code": providerCode}
			changed := tx.Model(&model.RealNameVerification{}).Where("id = ? AND status = ?", record.Id, model.RealNameVerificationStatusPending).Updates(updates)
			if changed.Error != nil || changed.RowsAffected == 0 {
				return changed.Error
			}
			if rewardQuota > 0 {
				if err := tx.Model(&model.User{}).Where("id = ?", record.UserId).Updates(map[string]any{"quota": gorm.Expr("quota + ?", rewardQuota), "gift_quota": gorm.Expr("gift_quota + ?", rewardQuota)}).Error; err != nil {
					return err
				}
				if err := tx.Model(&model.RealNameVerification{}).Where("id = ?", record.Id).Updates(map[string]any{"rewarded_at": now, "reward_quota": rewardQuota}).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		record.Status = model.RealNameVerificationStatusPassed
		record.VerifiedAt = &now
		if rewardQuota > 0 {
			record.RewardQuota = rewardQuota
			record.RewardedAt = &now
		}
		if err := publishRealNameVerificationSuccessMessage(record); err != nil {
			common.SysLog("\u5b9e\u540d\u8ba4\u8bc1\u6210\u529f\u7ad9\u5185\u6d88\u606f\u53d1\u9001\u5931\u8d25: " + err.Error())
		}
		return nil
	case "F":
		err = model.DB.Model(&model.RealNameVerification{}).Where("id = ? AND status = ?", record.Id, model.RealNameVerificationStatusPending).Updates(map[string]any{"status": model.RealNameVerificationStatusFailed, "provider_code": providerCode}).Error
		if err == nil {
			record.Status = model.RealNameVerificationStatusFailed
		}
		return err
	default:
		return nil
	}
}

func publishRealNameVerificationSuccessMessage(record *model.RealNameVerification) error {
	if record == nil || record.UserId <= 0 || record.VerifiedAt == nil {
		return errors.New("\u8ba4\u8bc1\u8bb0\u5f55\u65e0\u6548")
	}
	realNameVerificationMessageMutex.Lock()
	defer realNameVerificationMessageMutex.Unlock()
	title := "\u5b9e\u540d\u8ba4\u8bc1\u6210\u529f"
	content := "\u4f60\u7684\u5b9e\u540d\u8ba4\u8bc1\u5df2\u6210\u529f\u5b8c\u6210\uff0c\u8d26\u6237\u5df2\u901a\u8fc7\u963f\u91cc\u4e91\u91d1\u878d\u7ea7\u5b9e\u4eba\u8ba4\u8bc1\u3002"
	if record.RewardQuota > 0 {
		content += fmt.Sprintf("\u672c\u6b21\u8ba4\u8bc1\u5956\u52b1 %s \u5df2\u53d1\u653e\u81f3\u8d60\u9001\u4f59\u989d\u3002", strings.TrimSuffix(logger.LogQuotaManage(record.RewardQuota), " 额度"))
	}
	var existing model.UserMessage
	err := model.DB.Where(
		"receiver_user_id = ? AND biz_type = ? AND biz_id = ?",
		record.UserId, "real_name_verification", record.Id,
	).First(&existing).Error
	if err == nil {
		if existing.Title == title && existing.Content == content {
			return nil
		}
		return model.DB.Model(&existing).Updates(map[string]any{
			"title":   title,
			"content": content,
		}).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return PublishUserMessage(&model.UserMessage{
		ReceiverUserID: record.UserId,
		Type:           "real_name_verification_passed",
		Title:          title,
		Content:        content,
		BizType:        "real_name_verification",
		BizID:          record.Id,
	})
}

func newAliyunRealNameVerificationClient() (*cloudauth.Client, error) {
	if !setting.AliyunRealNameVerificationConfigured() {
		return nil, errors.New("实名认证服务暂未配置")
	}
	return cloudauth.NewClient(&openapi.Config{AccessKeyId: &setting.AliyunRealNameVerificationAccessKeyID, AccessKeySecret: &setting.AliyunRealNameVerificationAccessKeySecret, RegionId: &setting.AliyunRealNameVerificationRegionID, Endpoint: dara.String("cloudauth.aliyuncs.com")})
}

func toRealNameVerificationStatus(record *model.RealNameVerification) *RealNameVerificationStatus {
	status := &RealNameVerificationStatus{Status: record.Status, ExpiresAt: record.ExpiresAt.Unix(), RewardEnabled: setting.AliyunRealNameVerificationRewardEnabled, RewardAmount: setting.AliyunRealNameVerificationRewardAmount, RewardGranted: record.RewardedAt != nil}
	if record.RewardQuota > 0 {
		status.RewardAmount = float64(record.RewardQuota) / common.QuotaPerUnit
	}
	if record.VerifiedAt != nil {
		verifiedAt := record.VerifiedAt.Unix()
		status.VerifiedAt = &verifiedAt
	}
	return status
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func isAliyunInvalidCertNoError(code, message string) bool {
	normalizedCode := strings.TrimSpace(code)
	normalizedMessage := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(message), " ", ""))
	return (normalizedCode == "400" || normalizedCode == "401") &&
		(strings.Contains(normalizedMessage, "certno") || strings.Contains(normalizedMessage, "???"))
}

func buildAliyunRealNameVerificationURL(configuredURL, fallbackURL, launchToken string) (string, error) {
	targetURL := strings.TrimSpace(configuredURL)
	if targetURL == "" {
		targetURL = fallbackURL
	}
	parsedURL, err := url.Parse(targetURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", errors.New("必须是完整的 HTTP 或 HTTPS 地址")
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", errors.New("仅支持 HTTP 或 HTTPS 地址")
	}
	query := parsedURL.Query()
	query.Set("token", launchToken)
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}
