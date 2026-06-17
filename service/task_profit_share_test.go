package service

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/require"
)

func TestTryPostWalletProfitShareForTaskBilledQuota_SeedancePerVideoMarkup(t *testing.T) {
	require.NoError(t, model.DB.AutoMigrate(
		&model.AffInviteRelation{},
		&model.AffInviteProfitShareLog{},
	))

	oldMode := common.DistributorCommissionMode
	oldQuotaPerUnit := common.QuotaPerUnit
	oldVideoRules := ratio_setting.VideoPricingRules2JSONString()
	oldChannelVideoRules := ratio_setting.ChannelVideoPricingRules2JSONString()
	t.Cleanup(func() {
		common.DistributorCommissionMode = oldMode
		common.QuotaPerUnit = oldQuotaPerUnit
		require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(oldVideoRules))
		require.NoError(t, ratio_setting.UpdateChannelVideoPricingRulesByJSONString(oldChannelVideoRules))
		model.DB.Exec("DELETE FROM aff_invite_profit_share_logs")
		model.DB.Exec("DELETE FROM aff_invite_relations")
		model.DB.Exec("DELETE FROM channels")
		model.DB.Exec("DELETE FROM users")
	})

	common.DistributorCommissionMode = common.DistributorCommissionModeProfitShare
	common.QuotaPerUnit = 500000
	seedSeedancePerVideoRules(t)

	type laneCase struct {
		name         string
		input        string
		channelPrice float64
		globalPrice  float64
	}
	lanes := []laneCase{
		{name: "text_to_video", input: `{"model":"Seedance2.0","prompt":"test","size":"854x480","metadata":{"has_audio":false}}`, channelPrice: 1, globalPrice: 2},
		{name: "image_to_video", input: `{"model":"Seedance2.0","prompt":"test","image":"https://example.com/ref.jpg","size":"854x480","metadata":{"has_audio":false}}`, channelPrice: 2, globalPrice: 3},
		{name: "video_to_video", input: `{"model":"Seedance2.0","prompt":"test","input_reference":"https://example.com/ref.mp4","size":"854x480","metadata":{"has_audio":false}}`, channelPrice: 3, globalPrice: 4},
	}
	for _, lane := range lanes {
		for _, markup := range []float64{0, 30, 31} {
			t.Run(fmt.Sprintf("%s_%.0f", lane.name, markup), func(t *testing.T) {
				resetSeedanceProfitShareData(t, 0)

				billedQuota := int(math.Round(model.EffectiveRuleUnitPrice(lane.channelPrice, lane.globalPrice, 69, markup) * common.QuotaPerUnit))
				baseQuota := int(math.Round(model.EffectiveRuleUnitPrice(lane.channelPrice, lane.globalPrice, 69, 0) * common.QuotaPerUnit))
				markupSlice := billedQuota - baseQuota
				expectedReward := int(int64(markupSlice) * 9000 / 10000)
				markupSnapshot := markup

				task := &model.Task{
					TaskID:    "task_seedance_profit_share",
					UserId:    3,
					ChannelId: 2,
					Quota:     billedQuota,
					Group:     "default",
					Status:    model.TaskStatusSuccess,
					Properties: model.Properties{
						Input:           lane.input,
						OriginModelName: "Seedance2.0",
					},
					PrivateData: model.TaskPrivateData{
						BillingSource: BillingSourceWallet,
						BillingContext: &model.TaskBillingContext{
							GroupRatio:                  1,
							OriginModelName:             "Seedance2.0",
							ChannelPriceDiscountPercent: 69,
							MarkupDiscountPercent:       &markupSnapshot,
							VideoRuleUnit:               "per_video",
							VideoBillingMode:            lane.name,
							VideoChannelRulePrice:       lane.channelPrice,
							VideoGlobalRulePrice:        lane.globalPrice,
							VideoRuleWidth:              854,
							VideoRuleHeight:             480,
							VideoRuleHasAudio:           false,
						},
					},
				}

				TryPostWalletProfitShareForTaskBilledQuota(context.Background(), task, billedQuota, 0)

				var inviter model.User
				require.NoError(t, model.DB.Select("aff_quota", "aff_history").Where("id = ?", 2).First(&inviter).Error)
				require.Equal(t, expectedReward, inviter.AffQuota)
				require.Equal(t, expectedReward, inviter.AffHistoryQuota)

				var rel model.AffInviteRelation
				require.NoError(t, model.DB.Where("inviter_id = ? AND invitee_user_id = ?", 2, 3).First(&rel).Error)
				require.Equal(t, expectedReward, rel.ProfitShareEarnedQuota)

				var logs []model.AffInviteProfitShareLog
				require.NoError(t, model.DB.Order("id ASC").Find(&logs).Error)
				if expectedReward == 0 {
					require.Empty(t, logs)
					return
				}
				require.Len(t, logs, 1)
				require.Equal(t, billedQuota, logs[0].UserQuotaCharged)
				require.Equal(t, markupSlice, logs[0].MarkupSliceQuota)
				require.Equal(t, expectedReward, logs[0].RewardQuota)
				require.Equal(t, 9000, logs[0].CommissionBps)
			})
		}
	}
}

func seedSeedancePerVideoRules(t *testing.T) {
	t.Helper()
	channelRules := map[string]map[string]ratio_setting.VideoPricingRules{
		"2": {
			"Seedance2.0": {
				TextToVideoPerItem: []ratio_setting.VideoResolutionAudioPriceRule{
					{Resolution: "854x480", HasAudio: false, Price: 1},
					{Resolution: "854x480", HasAudio: true, Price: 1},
				},
				ImageToVideoPerItem: []ratio_setting.VideoResolutionAudioPriceRule{
					{Resolution: "854x480", HasAudio: false, Price: 2},
					{Resolution: "854x480", HasAudio: true, Price: 2},
				},
				VideoToVideoPerItem: []ratio_setting.VideoResolutionAudioPriceRule{
					{Resolution: "854x480", HasAudio: false, Price: 3},
					{Resolution: "854x480", HasAudio: true, Price: 3},
				},
			},
		},
	}
	globalRules := map[string]ratio_setting.VideoPricingRules{
		"Seedance2.0": {
			TextToVideoPerItem: []ratio_setting.VideoResolutionAudioPriceRule{
				{Resolution: "854x480", HasAudio: false, Price: 2},
				{Resolution: "854x480", HasAudio: true, Price: 2},
			},
			ImageToVideoPerItem: []ratio_setting.VideoResolutionAudioPriceRule{
				{Resolution: "854x480", HasAudio: false, Price: 3},
				{Resolution: "854x480", HasAudio: true, Price: 3},
			},
			VideoToVideoPerItem: []ratio_setting.VideoResolutionAudioPriceRule{
				{Resolution: "854x480", HasAudio: false, Price: 4},
				{Resolution: "854x480", HasAudio: true, Price: 4},
			},
		},
	}
	channelJSON, err := common.Marshal(channelRules)
	require.NoError(t, err)
	globalJSON, err := common.Marshal(globalRules)
	require.NoError(t, err)
	require.NoError(t, ratio_setting.UpdateChannelVideoPricingRulesByJSONString(string(channelJSON)))
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(string(globalJSON)))
}

func resetSeedanceProfitShareData(t *testing.T, markup float64) {
	t.Helper()
	model.DB.Exec("DELETE FROM aff_invite_profit_share_logs")
	model.DB.Exec("DELETE FROM aff_invite_relations")
	model.DB.Exec("DELETE FROM channels")
	model.DB.Exec("DELETE FROM users")

	costDiscount := 69.0
	defaultMarkup := 30.0
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:                   2,
		Name:                 "seedance2.0",
		Type:                 constant.ChannelTypeOpenAIVideo,
		Key:                  "sk-test",
		Status:               common.ChannelStatusEnabled,
		Models:               "Seedance2.0",
		RouteSlug:            "u2",
		PriceDiscountPercent: &costDiscount,
		MarkupDiscountRate:   &defaultMarkup,
	}).Error)
	require.NoError(t, model.DB.Create(&model.User{
		Id:                       2,
		Username:                 "test",
		Status:                   common.UserStatusEnabled,
		Role:                     common.RoleCommonUser,
		AffCode:                  "test-aff-code",
		IsDistributor:            1,
		DistributorCommissionBps: 9000,
	}).Error)
	require.NoError(t, model.DB.Create(&model.User{
		Id:        3,
		Username:  "test1",
		Status:    common.UserStatusEnabled,
		Role:      common.RoleCommonUser,
		AffCode:   "test1-aff-code",
		Quota:     409239936,
		InviterId: 2,
	}).Error)
	discounts, err := common.Marshal([]model.ModelMarkupDiscountRateUpdateRequest{
		{ModelName: "Seedance2.0", ChannelID: 2, MarkupDiscountRate: markup},
	})
	require.NoError(t, err)
	require.NoError(t, model.DB.Create(&model.AffInviteRelation{
		InviterId:               2,
		InviteeUserId:           3,
		CommissionRatioBps:      1000,
		ModelMarkupDiscountRate: string(discounts),
	}).Error)
}
