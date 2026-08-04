package controller

import (
	"bytes"
	"sort"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/xuri/excelize/v2"
)

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
	if value, err := f.GetCellValue(sheet, "D1"); err != nil || value != "调用折扣" {
		t.Fatalf("D1 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "E1"); err != nil || value != "折扣后价格" {
		t.Fatalf("E1 = %q, %v", value, err)
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
	if value, err := f.GetCellValue(sheet, "C3"); err != nil || value != "$0.04 / 次" {
		t.Fatalf("C3 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "E3"); err != nil || value != "$0.032 / 次" {
		t.Fatalf("E3 = %q, %v", value, err)
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
			want: "视频生成（按 token）\n无声\n文生视频 · 720p：$2 / 1M tokens",
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
			want: "视频生成（按秒）\n有声\n图生视频 · 1080p：$1.6 / 秒",
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
			want: "视频生成（按条）\n无声\n视频生视频 · 720p：$3 / 条",
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
			want: "视频生成（按条）\n通用\n文生视频 · 1080p：$4 / 条",
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
			},
		}
		want := "视频生成（按秒）\n" +
			"通用\n" +
			"文生视频 · 720p：$1 / 秒\n" +
			"文生视频 · 1080p：$2 / 秒\n" +
			"图生视频 · 480p：$0.5 / 秒"
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
			"无声\n" +
			"文生视频 · 720p：$1 / 秒\n" +
			"有声\n" +
			"文生视频 · 720p：$2 / 秒"
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
	spacedWant := "视频生成（按 token）\n无声\n文生视频 · 720p：$6.745 / 1M tokens"
	if got := formatDistributorModelDiscountVideoPricing(spaced, 1); got != spacedWant {
		t.Fatalf("Seedance 2.0 pricing = %q, want %q", got, spacedWant)
	}

	compactRules, ok := ratio_setting.GetVideoPricingRules("Seedance2.0")
	if !ok {
		t.Fatal("Seedance2.0 rules not found")
	}
	compact := model.InviteeModelMarkupDiscountRateItem{OfficialVideoPricingRules: &compactRules}
	compactWant := "视频生成（按条）\n有声\n文生视频 · 1080p：$9.5 / 条"
	if got := formatDistributorModelDiscountVideoPricing(compact, 1); got != compactWant {
		t.Fatalf("Seedance2.0 pricing = %q, want %q", got, compactWant)
	}
}
