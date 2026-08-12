package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

func setupAdminInviteeModelDiscountTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := model.DB
	dsn := fmt.Sprintf(
		"file:admin_invitee_model_discount_%d?mode=memory&cache=shared",
		time.Now().UnixNano(),
	)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	return db
}

func TestValidateAdminInviteeModelDiscountTargetChecksOwnership(t *testing.T) {
	db := setupAdminInviteeModelDiscountTestDB(t)
	previousMode := common.DistributorCommissionMode
	common.DistributorCommissionMode = common.DistributorCommissionModeProfitShare
	t.Cleanup(func() {
		common.DistributorCommissionMode = previousMode
	})

	distributor := model.User{
		Username:      "discount-owner",
		Password:      "password",
		AffCode:       "discount-owner-code",
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		IsDistributor: common.DistributorFlagYes,
	}
	if err := db.Create(&distributor).Error; err != nil {
		t.Fatalf("create distributor: %v", err)
	}
	invitee := model.User{
		Username:  "discount-invitee",
		Password:  "password",
		AffCode:   "discount-invitee-code",
		Role:      common.RoleCommonUser,
		Status:    common.UserStatusEnabled,
		InviterId: distributor.Id,
	}
	if err := db.Create(&invitee).Error; err != nil {
		t.Fatalf("create invitee: %v", err)
	}
	unrelated := model.User{
		Username: "discount-unrelated",
		Password: "password",
		AffCode:  "discount-unrelated-code",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	if err := db.Create(&unrelated).Error; err != nil {
		t.Fatalf("create unrelated user: %v", err)
	}

	if msg := validateAdminInviteeModelDiscountTarget(distributor.Id, invitee.Id); msg != "" {
		t.Fatalf("valid distributor invitee rejected: %s", msg)
	}
	if msg := validateAdminInviteeModelDiscountTarget(distributor.Id, unrelated.Id); msg == "" {
		t.Fatal("unrelated user was accepted as distributor invitee")
	}
	if msg := validateAdminInviteeModelDiscountTarget(unrelated.Id, invitee.Id); msg == "" {
		t.Fatal("ordinary user was accepted as distributor")
	}
}

func setDistributorModelDiscountExportCurrencyForTest(
	t *testing.T,
	displayType string,
	usdExchangeRate float64,
	customSymbol string,
	customExchangeRate float64,
) {
	t.Helper()
	setting := operation_setting.GetGeneralSetting()
	previousSetting := *setting
	previousUSDExchangeRate := operation_setting.USDExchangeRate
	t.Cleanup(func() {
		*setting = previousSetting
		operation_setting.USDExchangeRate = previousUSDExchangeRate
	})
	setting.QuotaDisplayType = displayType
	setting.CustomCurrencySymbol = customSymbol
	setting.CustomCurrencyExchangeRate = customExchangeRate
	operation_setting.USDExchangeRate = usdExchangeRate
}

func useDistributorModelDiscountUSDForTest(t *testing.T) {
	t.Helper()
	setDistributorModelDiscountExportCurrencyForTest(t, operation_setting.QuotaDisplayTypeUSD, 7, "¤", 1)
}

func TestExportDistributorModelDiscountTemplateAdminRequiresProfitShareMode(t *testing.T) {
	previousMode := common.DistributorCommissionMode
	common.DistributorCommissionMode = common.DistributorCommissionModeTopup
	t.Cleanup(func() {
		common.DistributorCommissionMode = previousMode
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/distributor/admin/model-discount-template/export", nil)
	ctx.Set("id", 1)

	ExportDistributorModelDiscountTemplateAdmin(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "当前站点未启用利润分成模式") {
		t.Fatalf("body = %q", body)
	}
}

func TestExportDistributorModelDiscountTemplateResponseWritesWorkbook(t *testing.T) {
	useDistributorModelDiscountUSDForTest(t)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/distributor/admin/model-discount-template/export", nil)

	exportDistributorModelDiscountTemplateResponse(ctx, []model.InviteeModelMarkupDiscountRateItem{{
		ModelName:                   "gpt-test",
		ChannelPath:                 "gpt-test/default",
		SupplierType:                "AIDC",
		ChannelPriceDiscountPercent: 80,
		OfficialBasePrice:           1,
		OfficialCompletionRatio:     1,
	}}, 0)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != inviteeModelDiscountExportContentType {
		t.Fatalf("content type = %q", contentType)
	}
	disposition := recorder.Header().Get("Content-Disposition")
	if !strings.HasPrefix(disposition, `attachment; filename="call-discount-`) || !strings.Contains(disposition, `.xlsx"; filename*=UTF-8''`) {
		t.Fatalf("content disposition = %q", disposition)
	}
	encodedFilename := strings.SplitN(disposition, "filename*=UTF-8''", 2)
	if len(encodedFilename) != 2 {
		t.Fatalf("missing RFC 5987 filename*: %q", disposition)
	}
	decodedFilename, err := url.PathUnescape(encodedFilename[1])
	if err != nil {
		t.Fatalf("decode filename*: %v", err)
	}
	if !strings.HasPrefix(decodedFilename, "调用折扣-") || !strings.HasSuffix(decodedFilename, ".xlsx") {
		t.Fatalf("decoded filename* = %q", decodedFilename)
	}
	workbook, err := excelize.OpenReader(bytes.NewReader(recorder.Body.Bytes()))
	if err != nil {
		t.Fatalf("response is not an xlsx workbook: %v", err)
	}
	defer workbook.Close()
}

func TestBuildDistributorModelDiscountTemplateExportWorkbook(t *testing.T) {
	useDistributorModelDiscountUSDForTest(t)
	cacheRatio := 0.1
	createCacheRatio := 1.25
	data, err := buildDistributorModelDiscountTemplateExportWorkbook(
		[]model.InviteeModelMarkupDiscountRateItem{
			{
				ModelName:                   "gpt-4.1",
				ChannelPath:                 "gpt-4.1/primary",
				SupplierType:                "AIDC",
				ChannelPriceDiscountPercent: 72.34,
				OfficialBasePrice:           1.5,
				ChannelBasePrice:            99,
				DefaultMarkupDiscountRate:   25,
				CurrentMarkupDiscountRate:   50,
				OfficialPricingQuotaType:    0,
				OfficialCompletionRatio:     4,
				OfficialCacheRatio:          &cacheRatio,
				OfficialCreateCacheRatio:    &createCacheRatio,
			},
			{
				ModelName:                   "dall-e-3",
				ChannelPath:                 "P1/dall-e-3/001",
				SupplierType:                "公有云",
				ChannelPriceDiscountPercent: 80,
				OfficialBasePrice:           0.04,
				ChannelBasePrice:            9,
				DefaultMarkupDiscountRate:   20,
				CurrentMarkupDiscountRate:   40,
				OfficialPricingQuotaType:    1,
			},
			{
				ModelName:                   "tier-model",
				ChannelPath:                 "tier-model/tier-route",
				SupplierType:                "企业中转站",
				ChannelPriceDiscountPercent: 50,
				OfficialPricingQuotaType:    3,
				OfficialRequestTierPricing: &ratio_setting.RequestTierPricing{
					Currency: ratio_setting.RequestTierCurrencyUSD,
					Boundary: ratio_setting.RequestTierBoundaryLt,
					Tiers: []ratio_setting.RequestTierBand{
						{
							UpTo: 32000,
							Prices: ratio_setting.RequestTierPrices{
								Input: 2, Output: 8, CacheRead: 0.2, CacheWrite: 2.5,
							},
						},
						{
							UpTo: 0,
							Prices: ratio_setting.RequestTierPrices{
								Input: 3, Output: 12, CacheRead: 0.3, CacheWrite: 3.75,
							},
						},
					},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("build workbook: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	defer f.Close()

	const sheet = "调用折扣"
	if value, err := f.GetCellValue(sheet, "A1"); err != nil || value != "模型 / 通道路径" {
		t.Fatalf("A1 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "B1"); err != nil || value != "模型类型" {
		t.Fatalf("B1 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "C1"); err != nil || value != "官方价格（全局价）" {
		t.Fatalf("C1 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "D1"); err != nil || value != "代理调用折扣" {
		t.Fatalf("D1 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "E1"); err != nil || value != "代理折扣后价格" {
		t.Fatalf("E1 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "F1"); err != nil || value != "平台折扣" {
		t.Fatalf("F1 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "G1"); err != nil || value != "平台折扣后价格" {
		t.Fatalf("G1 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "A2"); err != nil || value != "gpt-4.1\ngpt-4.1/primary" {
		t.Fatalf("A2 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "B2"); err != nil || value != "AIDC" {
		t.Fatalf("B2 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "C2"); err != nil || value != "文本按量\n输入 $3 / 1M tokens；输出 $12 / 1M tokens；缓存读 $0.3 / 1M tokens；缓存写 $3.75 / 1M tokens" {
		t.Fatalf("C2 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "D2"); err != nil || value != "72.3%" {
		t.Fatalf("D2 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "E2"); err != nil || value != "文本按量\n输入 $2.1702 / 1M tokens；输出 $8.6808 / 1M tokens；缓存读 $0.21702 / 1M tokens；缓存写 $2.71275 / 1M tokens" {
		t.Fatalf("E2 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "F2"); err != nil || value != "97.3%" {
		t.Fatalf("F2 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "G2"); err != nil || value != "文本按量\n输入 $2.9202 / 1M tokens；输出 $11.6808 / 1M tokens；缓存读 $0.29202 / 1M tokens；缓存写 $3.65025 / 1M tokens" {
		t.Fatalf("G2 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "C3"); err != nil || value != "$0.04 / 次" {
		t.Fatalf("C3 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "E3"); err != nil || value != "$0.032 / 次" {
		t.Fatalf("E3 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "F3"); err != nil || value != "100.0%" {
		t.Fatalf("F3 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "G3"); err != nil || value != "$0.04 / 次" {
		t.Fatalf("G3 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "C4"); err != nil || value != "阶梯价\n0 ≤ 输入 token < 32000：输入 $2 / 1M tokens；输出 $8 / 1M tokens；缓存读 $0.2 / 1M tokens；缓存写 $2.5 / 1M tokens\n输入 token ≥ 32000：输入 $3 / 1M tokens；输出 $12 / 1M tokens；缓存读 $0.3 / 1M tokens；缓存写 $3.75 / 1M tokens" {
		t.Fatalf("C4 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "E4"); err != nil || value != "阶梯价\n0 ≤ 输入 token < 32000：输入 $1 / 1M tokens；输出 $4 / 1M tokens；缓存读 $0.1 / 1M tokens；缓存写 $1.25 / 1M tokens\n输入 token ≥ 32000：输入 $1.5 / 1M tokens；输出 $6 / 1M tokens；缓存读 $0.15 / 1M tokens；缓存写 $1.875 / 1M tokens" {
		t.Fatalf("E4 = %q, %v", value, err)
	}
}

func TestFormatDistributorModelDiscountImagePricing(t *testing.T) {
	useDistributorModelDiscountUSDForTest(t)
	imageRatio := 0.5
	fallbackImagePrice := 0.99
	item := model.InviteeModelMarkupDiscountRateItem{
		OfficialModelRatio: 1.5,
		OfficialImageRatio: &imageRatio,
		OfficialImagePrice: &fallbackImagePrice,
		OfficialImagePricingRules: &ratio_setting.ImagePricingRules{
			TextToImagePerImage: []ratio_setting.ImageResolutionPerImageRule{
				{Resolution: "1K", ImagePrice: 0.02},
				{Resolution: "2K", ImagePrice: 0.04},
			},
			ImageToImagePerImage: []ratio_setting.ImageResolutionPerImageRule{
				{Resolution: "1K", ImagePrice: 0.03},
			},
		},
	}

	official := formatDistributorModelDiscountImagePricing(item, 1)
	wantOfficial := "图片生成\n" +
		"文生图 · 1K：$0.02 / 张\n" +
		"文生图 · 2K：$0.04 / 张\n" +
		"图生图 · 1K：$0.03 / 张\n" +
		"图片输入：$1.5 / 1M tokens"
	if official != wantOfficial {
		t.Fatalf("official image pricing = %q, want %q", official, wantOfficial)
	}

	discounted := formatDistributorModelDiscountImagePricing(item, 0.5)
	wantDiscounted := "图片生成\n" +
		"文生图 · 1K：$0.01 / 张\n" +
		"文生图 · 2K：$0.02 / 张\n" +
		"图生图 · 1K：$0.015 / 张\n" +
		"图片输入：$0.75 / 1M tokens"
	if discounted != wantDiscounted {
		t.Fatalf("discounted image pricing = %q, want %q", discounted, wantDiscounted)
	}
}

func TestFormatDistributorModelDiscountOfficialPriceCombinesTextImageVideo(t *testing.T) {
	useDistributorModelDiscountUSDForTest(t)
	imagePrice := 0.1
	videoPrice := 0.2
	item := model.InviteeModelMarkupDiscountRateItem{
		OfficialBasePrice:        1,
		OfficialModelRatio:       1,
		OfficialPricingQuotaType: 0,
		OfficialCompletionRatio:  2,
		OfficialImagePrice:       &imagePrice,
		OfficialVideoPrice:       &videoPrice,
	}

	got := formatDistributorModelDiscountOfficialPrice(item, 0.5)
	want := "文本按量\n" +
		"输入 $1 / 1M tokens；输出 $2 / 1M tokens\n" +
		"图片生成：$0.05 / 张\n" +
		"视频生成：$0.1 / 条"
	if got != want {
		t.Fatalf("combined pricing = %q, want %q", got, want)
	}
}

func TestDistributorModelDiscountPlatformPriceAndCombinedDiscount(t *testing.T) {
	useDistributorModelDiscountUSDForTest(t)
	completion := 2.0
	item := model.InviteeModelMarkupDiscountRateItem{
		PlatformPricing: &model.PricingAPIItem{
			Pricing: model.Pricing{ModelRatio: 1, CompletionRatio: &completion},
			ChannelList: []model.PricingChannelItem{{
				ModelRatio: 0.5, CompletionRatio: 3, EffectiveCostPercent: 80, MarkupDiscountRate: 10,
			}},
		},
	}
	if got, want := formatDistributorModelDiscountPlatformPrice(item, 1.5), "文本按量\n输入 $1.5 / 1M tokens；输出 $4.2 / 1M tokens"; got != want {
		t.Fatalf("platform token price = %q, want %q", got, want)
	}
	// 平台折扣为代理调用折扣与默认加价折扣之和。
	item.ChannelPriceDiscountPercent = 72.34
	item.DefaultMarkupDiscountRate = 10
	if got, want := formatInviteeModelDiscountMarkupRate(item.ChannelPriceDiscountPercent+item.DefaultMarkupDiscountRate), "82.3%"; got != want {
		t.Fatalf("platform call discount = %q, want %q", got, want)
	}

	requestItem := model.InviteeModelMarkupDiscountRateItem{
		PlatformPricing: &model.PricingAPIItem{
			Pricing: model.Pricing{ModelPrice: 1},
			ChannelList: []model.PricingChannelItem{{
				QuotaType: 1, ModelPrice: 0.5, EffectiveCostPercent: 80, MarkupDiscountRate: 10,
			}},
		},
	}
	if got, want := formatDistributorModelDiscountPlatformPrice(requestItem, 1.5), "$0.75 / 次"; got != want {
		t.Fatalf("platform request price = %q, want %q", got, want)
	}
	requestItem.ChannelPriceDiscountPercent = 80
	requestItem.DefaultMarkupDiscountRate = 10
	if got, want := formatInviteeModelDiscountMarkupRate(requestItem.ChannelPriceDiscountPercent+requestItem.DefaultMarkupDiscountRate), "90.0%"; got != want {
		t.Fatalf("platform request call discount = %q, want %q", got, want)
	}
}

func TestDistributorModelDiscountAudioPricing(t *testing.T) {
	useDistributorModelDiscountUSDForTest(t)
	audioRatio := 0.5
	audioCompletionRatio := 3.0
	channelOnlyAudioRatio := 9.0
	channelOnlyAudioCompletionRatio := 8.0
	item := model.InviteeModelMarkupDiscountRateItem{
		OfficialModelRatio:           2,
		OfficialAudioRatio:           &audioRatio,
		OfficialAudioCompletionRatio: &audioCompletionRatio,
		PlatformPricing: &model.PricingAPIItem{
			Pricing: model.Pricing{
				ModelRatio:           2,
				AudioRatio:           &audioRatio,
				AudioCompletionRatio: &audioCompletionRatio,
			},
			ChannelList: []model.PricingChannelItem{{
				ModelRatio:                 1,
				EffectiveCostPercent:       80,
				MarkupDiscountRate:         10,
				OptionAudioRatio:           &channelOnlyAudioRatio,
				OptionAudioCompletionRatio: &channelOnlyAudioCompletionRatio,
			}},
		},
	}
	if got := formatDistributorModelDiscountOfficialPrice(item, 1); !strings.Contains(got, "音频输入 $2 / 1M tokens") || !strings.Contains(got, "音频输出 $6 / 1M tokens") {
		t.Fatalf("official audio pricing = %q", got)
	}
	if got := formatDistributorModelDiscountOfficialPrice(item, 0.5); !strings.Contains(got, "音频输入 $1 / 1M tokens") || !strings.Contains(got, "音频输出 $3 / 1M tokens") {
		t.Fatalf("agent audio pricing = %q", got)
	}
	if got := formatDistributorModelDiscountPlatformPrice(item, 1.5); !strings.Contains(got, "音频输入 $1.5 / 1M tokens") || !strings.Contains(got, "音频输出 $4.5 / 1M tokens") {
		t.Fatalf("platform audio must use global audio ratios after effective input rate: %q", got)
	}

	onlyInputAudioRatio := 0.25
	onlyInput := model.InviteeModelMarkupDiscountRateItem{
		OfficialModelRatio: 1,
		OfficialAudioRatio: &onlyInputAudioRatio,
		PlatformPricing: &model.PricingAPIItem{
			Pricing:     model.Pricing{ModelRatio: 1, AudioRatio: &onlyInputAudioRatio},
			ChannelList: []model.PricingChannelItem{{ModelRatio: 0.5, EffectiveCostPercent: 100}},
		},
	}
	if got := formatDistributorModelDiscountPlatformPrice(onlyInput, 1); !strings.Contains(got, "音频输入") || strings.Contains(got, "音频输出") {
		t.Fatalf("input-only audio pricing = %q", got)
	}
}

func TestDistributorModelDiscountPlatformTierAndMediaPricesUseGroupRatio(t *testing.T) {
	useDistributorModelDiscountUSDForTest(t)
	imageTier := model.ImagePerImageTierRow{UsdAfterChannelDiscount: 0.5, UsdOfficial: 1, Resolution: "1K", Lane: "text_to_image"}
	videoTier := model.VideoFlatClipTierRow{UsdAfterChannelDiscount: 0.5, UsdOfficial: 1, Resolution: "720p", Lane: "text_to_video"}
	item := model.InviteeModelMarkupDiscountRateItem{
		OfficialRequestTierPricing: &ratio_setting.RequestTierPricing{Tiers: []ratio_setting.RequestTierBand{{Prices: ratio_setting.RequestTierPrices{Input: 1, Output: 1, CacheRead: 1, CacheWrite: 1}}}},
		PlatformPricing: &model.PricingAPIItem{
			Pricing: model.Pricing{},
			ChannelList: []model.PricingChannelItem{{
				QuotaType:            3,
				RequestTierPricing:   ratio_setting.RequestTierPricing{Tiers: []ratio_setting.RequestTierBand{{Prices: ratio_setting.RequestTierPrices{Input: 0.5, Output: 0.5, CacheRead: 0.5, CacheWrite: 0.5}}}},
				EffectiveCostPercent: 100,
			}},
			ImagePerImageHint: &model.ImagePerImagePricingHint{Tiers: []model.ImagePerImageTierRow{imageTier}},
		},
	}
	if got := formatDistributorModelDiscountPlatformPrice(item, 1.5); !strings.Contains(got, "$0.75 / 张") {
		t.Fatalf("tier/image platform prices = %q", got)
	}

	videoItem := item
	videoItem.PlatformPricing.VideoFlatClipHint = &model.VideoFlatClipPricingHint{BillingMode: "per_item", Tiers: []model.VideoFlatClipTierRow{videoTier}}
	if got := formatDistributorModelDiscountPlatformPrice(videoItem, 1.5); !strings.Contains(got, "$0.75 / 条") || strings.Contains(got, "阶梯价") {
		t.Fatalf("video model must hide text tiers and keep video price: %q", got)
	}
}

func TestDistributorModelDiscountPlatformSimpleVideoMatchesSideSheet(t *testing.T) {
	useDistributorModelDiscountUSDForTest(t)
	videoPrice := 2.0
	channelVideoPrice := 1.0
	item := model.InviteeModelMarkupDiscountRateItem{
		PlatformPricing: &model.PricingAPIItem{
			Pricing: model.Pricing{ModelRatio: 1, VideoPrice: &videoPrice},
			ChannelList: []model.PricingChannelItem{{
				ModelRatio: 1, EffectiveCostPercent: 10, MarkupDiscountRate: 90, OptionVideoPrice: &channelVideoPrice,
			}},
		},
	}
	// The simple-video row uses channel_video_price directly: it does not apply
	// cost/markup again. Its display price uses the selected group ratio.
	if got, want := formatDistributorModelDiscountPlatformPrice(item, 1.5), "视频生成：$1.5 / 条"; got != want {
		t.Fatalf("simple video platform price = %q, want %q", got, want)
	}
	imagePrice := 0.1
	item.OfficialImagePrice = &imagePrice
	item.PlatformPricing.VideoPrice = nil
	if got := formatDistributorModelDiscountPlatformPrice(item, 1); strings.Contains(got, "图片生成") {
		t.Fatalf("simple image price is not rendered by the side sheet: %q", got)
	}
}

func TestDistributorModelDiscountPlatformVideoUsesAgentColumnModeFirstLayout(t *testing.T) {
	useDistributorModelDiscountUSDForTest(t)
	silent := false
	audio := true
	item := model.InviteeModelMarkupDiscountRateItem{
		PlatformPricing: &model.PricingAPIItem{
			Pricing:     model.Pricing{},
			ChannelList: []model.PricingChannelItem{{}},
			VideoFlatClipHint: &model.VideoFlatClipPricingHint{
				BillingMode: "per_item",
				Tiers: []model.VideoFlatClipTierRow{
					// nil is the hint's already-collapsed same-price audio pair.
					{UsdAfterChannelDiscount: 1, Resolution: "720p", Lane: "text_to_video"},
					{UsdAfterChannelDiscount: 2, Resolution: "1080p", Lane: "image_to_video", HasAudio: &silent},
					{UsdAfterChannelDiscount: 3, Resolution: "1080p", Lane: "image_to_video", HasAudio: &audio},
					// Be defensive if an uncollapsed equal-price pair reaches export.
					{UsdAfterChannelDiscount: 4, Resolution: "480p", Lane: "video_to_video", HasAudio: &silent},
					{UsdAfterChannelDiscount: 4, Resolution: "480p", Lane: "video_to_video", HasAudio: &audio},
					{UsdAfterChannelDiscount: 5, Resolution: "720p", Lane: "video_to_video_input_legacy"},
				},
			},
		},
	}
	got := formatDistributorModelDiscountPlatformPrice(item, 1)
	want := "视频生成（按条）\n" +
		"文生视频\n" +
		"720p：$1 / 条\n" +
		"图生视频\n" +
		"无声\n" +
		"1080p：$2 / 条\n" +
		"有声\n" +
		"1080p：$3 / 条\n" +
		"视频生视频\n" +
		"480p：$4 / 条\n" +
		"视频生视频（输入）\n" +
		"720p：$5 / 条"
	if got != want {
		t.Fatalf("platform video categories = %q, want %q", got, want)
	}
}

func TestDistributorModelDiscountASRPricingMatchesSideSheet(t *testing.T) {
	useDistributorModelDiscountUSDForTest(t)
	asrPrice := 0.1234567
	item := model.InviteeModelMarkupDiscountRateItem{
		OfficialASRPrice: &asrPrice,
		PlatformPricing: &model.PricingAPIItem{
			Pricing:     model.Pricing{ModelRatio: 9, ASRPrice: &asrPrice},
			ChannelList: []model.PricingChannelItem{{ModelRatio: 9}},
		},
	}
	if got, want := formatDistributorModelDiscountOfficialPrice(item, 1), "语音识别：$0.123456 / 秒"; got != want {
		t.Fatalf("official ASR price = %q, want %q", got, want)
	}
	if got, want := formatDistributorModelDiscountOfficialPrice(item, 0.5), "语音识别：$0.061728 / 秒"; got != want {
		t.Fatalf("agent ASR price = %q, want %q", got, want)
	}
	if got, want := formatDistributorModelDiscountPlatformPrice(item, 1.5), "语音识别：$0.185185 / 秒"; got != want {
		t.Fatalf("platform ASR price = %q, want %q", got, want)
	}
}

func TestFormatDistributorModelDiscountVideoPricingPriorityAndUnits(t *testing.T) {
	useDistributorModelDiscountUSDForTest(t)
	videoPrice := 5.0
	videoRatio := 2.0
	tests := []struct {
		name string
		item model.InviteeModelMarkupDiscountRateItem
		want string
	}{
		{
			name: "per token rules override lower priorities",
			item: model.InviteeModelMarkupDiscountRateItem{
				OfficialVideoPricingRules: &ratio_setting.VideoPricingRules{
					TextToVideoPerToken: []ratio_setting.VideoResolutionAudioPriceRule{
						{Resolution: "720p", HasAudio: false, Price: 4},
					},
					TextToVideoPerSecond: []ratio_setting.VideoResolutionAudioPriceRule{
						{Resolution: "720p", HasAudio: false, Price: 2},
					},
					TextToVideoPerItem: []ratio_setting.VideoResolutionAudioPriceRule{
						{Resolution: "720p", HasAudio: false, Price: 10},
					},
				},
			},
			want: "视频生成（按 token）\n文生视频\n无声\n720p：$2 / 1M tokens",
		},
		{
			name: "per second",
			item: model.InviteeModelMarkupDiscountRateItem{
				OfficialVideoPricingRules: &ratio_setting.VideoPricingRules{
					ImageToVideoPerSecond: []ratio_setting.VideoResolutionAudioPriceRule{
						{Resolution: "1080p", HasAudio: true, Price: 3.2},
					},
				},
			},
			want: "视频生成（按秒）\n图生视频\n有声\n1080p：$1.6 / 秒",
		},
		{
			name: "per item",
			item: model.InviteeModelMarkupDiscountRateItem{
				OfficialVideoPricingRules: &ratio_setting.VideoPricingRules{
					VideoToVideoPerItem: []ratio_setting.VideoResolutionAudioPriceRule{
						{Resolution: "720p", HasAudio: false, Price: 6},
					},
				},
			},
			want: "视频生成（按条）\n视频生视频\n无声\n720p：$3 / 条",
		},
		{
			name: "legacy per video",
			item: model.InviteeModelMarkupDiscountRateItem{
				OfficialVideoPricingRules: &ratio_setting.VideoPricingRules{
					TextToVideoPerVideo: []ratio_setting.VideoResolutionPerVideoRule{
						{Resolution: "1080p", VideoPrice: 8},
					},
				},
			},
			want: "视频生成（按条）\n文生视频\n1080p：$4 / 条",
		},
		{
			name: "legacy video input and output keep separate mode blocks",
			item: model.InviteeModelMarkupDiscountRateItem{
				OfficialVideoPricingRules: &ratio_setting.VideoPricingRules{
					VideoToVideoInputPerVideo: []ratio_setting.VideoResolutionPerVideoRule{
						{Resolution: "720p", VideoPrice: 6},
					},
					VideoToVideoOutputPerVideo: []ratio_setting.VideoResolutionPerVideoRule{
						{Resolution: "1080p", VideoPrice: 10},
					},
				},
			},
			want: "视频生成（按条）\n视频生视频（输入）\n720p：$3 / 条\n视频生视频（输出）\n1080p：$5 / 条",
		},
		{
			name: "video price precedes legacy ratio",
			item: model.InviteeModelMarkupDiscountRateItem{
				OfficialModelRatio:           1.5,
				OfficialVideoPrice:           &videoPrice,
				OfficialVideoRatio:           &videoRatio,
				OfficialVideoCompletionRatio: 3,
			},
			want: "视频生成：$2.5 / 条",
		},
		{
			name: "legacy token ratios",
			item: model.InviteeModelMarkupDiscountRateItem{
				OfficialModelRatio:           1.5,
				OfficialVideoRatio:           &videoRatio,
				OfficialVideoCompletionRatio: 3,
			},
			want: "视频 token：输入 $3 / 1M tokens；输出 $9 / 1M tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDistributorModelDiscountVideoPricing(tt.item, 0.5)
			if got != tt.want {
				t.Fatalf("video pricing = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatDistributorModelDiscountUnitPriceMatchesHomepageCurrency(t *testing.T) {
	tests := []struct {
		name               string
		displayType        string
		usdExchangeRate    float64
		customSymbol       string
		customExchangeRate float64
		want               string
	}{
		{
			name:        "USD",
			displayType: operation_setting.QuotaDisplayTypeUSD,
			want:        "$3 / 1M tokens",
		},
		{
			name:            "CNY",
			displayType:     operation_setting.QuotaDisplayTypeCNY,
			usdExchangeRate: 7,
			want:            "¥21 / 1M tokens",
		},
		{
			name:               "custom",
			displayType:        operation_setting.QuotaDisplayTypeCustom,
			customSymbol:       "₱",
			customExchangeRate: 2.5,
			want:               "₱7.5 / 1M tokens",
		},
		{
			name:        "tokens falls back to USD",
			displayType: operation_setting.QuotaDisplayTypeTokens,
			want:        "$3 / 1M tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setDistributorModelDiscountExportCurrencyForTest(
				t,
				tt.displayType,
				tt.usdExchangeRate,
				tt.customSymbol,
				tt.customExchangeRate,
			)
			if got := formatDistributorModelDiscountUnitPrice(3, "1M tokens"); got != tt.want {
				t.Fatalf("unit price = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatDistributorModelDiscountPriceTruncatesToSixDecimals(t *testing.T) {
	useDistributorModelDiscountUSDForTest(t)
	tests := []struct {
		value float64
		want  string
	}{
		{value: 6.745, want: "$6.745 / 次"},
		{value: 0.019999, want: "$0.019999 / 次"},
		{value: 0.0199999, want: "$0.019999 / 次"},
		{value: 0.6, want: "$0.6 / 次"},
	}
	for _, tt := range tests {
		if got := formatDistributorModelDiscountUnitPrice(tt.value, "次"); got != tt.want {
			t.Fatalf("unit price for %.9f = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestFormatDistributorModelDiscountPriceTruncatesAfterCNYConversion(t *testing.T) {
	setDistributorModelDiscountExportCurrencyForTest(t, operation_setting.QuotaDisplayTypeCNY, 7, "¤", 1)
	if got := formatDistributorModelDiscountUnitPrice(0.01999999, "次"); got != "¥0.139999 / 次" {
		t.Fatalf("CNY unit price = %q, want %q", got, "¥0.139999 / 次")
	}
}

func TestFormatDistributorModelDiscountVideoPricingGroupsAudioPrices(t *testing.T) {
	useDistributorModelDiscountUSDForTest(t)

	t.Run("same prices merge into general and sort resolutions", func(t *testing.T) {
		item := model.InviteeModelMarkupDiscountRateItem{
			OfficialVideoPricingRules: &ratio_setting.VideoPricingRules{
				TextToVideoPerSecond: []ratio_setting.VideoResolutionAudioPriceRule{
					{Resolution: "1080p", HasAudio: true, Price: 2},
					{Resolution: "720p", HasAudio: false, Price: 1},
					{Resolution: "1080p", HasAudio: false, Price: 2},
					{Resolution: "720p", HasAudio: true, Price: 1},
				},
				ImageToVideoPerSecond: []ratio_setting.VideoResolutionAudioPriceRule{
					{Resolution: "480p", HasAudio: false, Price: 0.5},
					{Resolution: "480p", HasAudio: true, Price: 0.5},
				},
				VideoToVideoPerSecond: []ratio_setting.VideoResolutionAudioPriceRule{
					{Resolution: "360p", HasAudio: false, Price: 0.25},
					{Resolution: "360p", HasAudio: true, Price: 0.25},
				},
			},
		}
		want := "视频生成（按秒）\n" +
			"文生视频\n" +
			"720p：$1 / 秒\n" +
			"1080p：$2 / 秒\n" +
			"图生视频\n" +
			"480p：$0.5 / 秒\n" +
			"视频生视频\n" +
			"360p：$0.25 / 秒"
		if got := formatDistributorModelDiscountVideoPricing(item, 1); got != want {
			t.Fatalf("same-price video pricing = %q, want %q", got, want)
		}
	})

	t.Run("different prices split silent before audio", func(t *testing.T) {
		item := model.InviteeModelMarkupDiscountRateItem{
			OfficialVideoPricingRules: &ratio_setting.VideoPricingRules{
				TextToVideoPerSecond: []ratio_setting.VideoResolutionAudioPriceRule{
					{Resolution: "720p", HasAudio: true, Price: 2},
					{Resolution: "720p", HasAudio: false, Price: 1},
				},
			},
		}
		want := "视频生成（按秒）\n" +
			"文生视频\n" +
			"无声\n" +
			"720p：$1 / 秒\n" +
			"有声\n" +
			"720p：$2 / 秒"
		if got := formatDistributorModelDiscountVideoPricing(item, 1); got != want {
			t.Fatalf("split video pricing = %q, want %q", got, want)
		}
	})
}

func TestDistributorModelDiscountResolutionNaturalSort(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   []string
	}{
		{
			name:   "p resolutions",
			values: []string{"1080p", "480p", "720p"},
			want:   []string{"480p", "720p", "1080p"},
		},
		{
			name:   "k resolutions",
			values: []string{"4K", "1K", "2K"},
			want:   []string{"1K", "2K", "4K"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := append([]string(nil), tt.values...)
			sort.SliceStable(got, func(i, j int) bool {
				return distributorModelDiscountResolutionLess(got[i], got[j])
			})
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("sorted resolutions = %#v, want %#v", got, tt.want)
				}
			}
		})
	}
}

func TestFormatDistributorModelDiscountTierPricingUsesSystemDisplayCurrency(t *testing.T) {
	setDistributorModelDiscountExportCurrencyForTest(t, operation_setting.QuotaDisplayTypeCNY, 7, "¤", 1)
	rule := &ratio_setting.RequestTierPricing{
		Currency: ratio_setting.RequestTierCurrencyCNY,
		Tiers: []ratio_setting.RequestTierBand{
			{
				UpTo: 0,
				Prices: ratio_setting.RequestTierPrices{
					Input: 14, Output: 28, CacheRead: 1.4, CacheWrite: 17.5,
				},
			},
		},
	}
	want := "阶梯价\n输入 token ≥ 0：输入 ¥7 / 1M tokens；输出 ¥14 / 1M tokens；缓存读 ¥0.7 / 1M tokens；缓存写 ¥8.75 / 1M tokens"
	if got := formatDistributorModelDiscountTierPricing(rule, 0.5); got != want {
		t.Fatalf("tier pricing = %q, want %q", got, want)
	}
}

func TestFormatDistributorModelDiscountTierPricingConvertsUSDToSystemCurrency(t *testing.T) {
	setDistributorModelDiscountExportCurrencyForTest(t, operation_setting.QuotaDisplayTypeCNY, 7, "¤", 1)
	rule := &ratio_setting.RequestTierPricing{
		Currency: ratio_setting.RequestTierCurrencyUSD,
		Tiers: []ratio_setting.RequestTierBand{
			{
				Prices: ratio_setting.RequestTierPrices{
					Input: 3, Output: 4, CacheRead: 0.5, CacheWrite: 1.25,
				},
			},
		},
	}
	want := "阶梯价\n输入 token ≥ 0：输入 ¥21 / 1M tokens；输出 ¥28 / 1M tokens；缓存读 ¥3.5 / 1M tokens；缓存写 ¥8.75 / 1M tokens"
	if got := formatDistributorModelDiscountTierPricing(rule, 1); got != want {
		t.Fatalf("tier pricing = %q, want %q", got, want)
	}
}

func TestDistributorModelDiscountVideoRulesKeepExactModelNames(t *testing.T) {
	useDistributorModelDiscountUSDForTest(t)
	previous := ratio_setting.VideoPricingRules2JSONString()
	t.Cleanup(func() {
		if err := ratio_setting.UpdateVideoPricingRulesByJSONString(previous); err != nil {
			t.Fatalf("restore video pricing rules: %v", err)
		}
	})

	const rulesJSON = `{
		"Seedance 2.0": {
			"text_to_video_per_token": [
				{"resolution":"720p","has_audio":false,"price":6.745}
			]
		},
		"Seedance2.0": {
			"text_to_video_per_item": [
				{"resolution":"1080p","has_audio":true,"price":9.5}
			]
		}
	}`
	if err := ratio_setting.UpdateVideoPricingRulesByJSONString(rulesJSON); err != nil {
		t.Fatalf("set video pricing rules: %v", err)
	}

	spacedRules, ok := ratio_setting.GetVideoPricingRules("Seedance 2.0")
	if !ok {
		t.Fatal("Seedance 2.0 rules not found")
	}
	spaced := model.InviteeModelMarkupDiscountRateItem{OfficialVideoPricingRules: &spacedRules}
	spacedWant := "视频生成（按 token）\n文生视频\n无声\n720p：$6.745 / 1M tokens"
	if got := formatDistributorModelDiscountVideoPricing(spaced, 1); got != spacedWant {
		t.Fatalf("Seedance 2.0 pricing = %q, want %q", got, spacedWant)
	}

	compactRules, ok := ratio_setting.GetVideoPricingRules("Seedance2.0")
	if !ok {
		t.Fatal("Seedance2.0 rules not found")
	}
	compact := model.InviteeModelMarkupDiscountRateItem{OfficialVideoPricingRules: &compactRules}
	compactWant := "视频生成（按条）\n文生视频\n有声\n1080p：$9.5 / 条"
	if got := formatDistributorModelDiscountVideoPricing(compact, 1); got != compactWant {
		t.Fatalf("Seedance2.0 pricing = %q, want %q", got, compactWant)
	}
}
