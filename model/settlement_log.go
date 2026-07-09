package model

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// SettlementExportFilter 结算单导出筛选。
type SettlementExportFilter struct {
	AdminLogExportFilter
	UserIDs     []int
	InviterIDs  []int
	ChannelIDs  []int
}

// GetSettlementLogsForExport 拉取结算单导出日志（仅消费类，升序）。
func GetSettlementLogsForExport(filter SettlementExportFilter) ([]*Log, int64, error) {
	tx := applySettlementExportFilter(LOG_DB, filter)

	var total int64
	if err := tx.Model(&Log{}).Limit(logExportCountLimit).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total > logExportCountLimit {
		return nil, total, fmt.Errorf("导出行数超过 %d 上限", logExportCountLimit)
	}

	var logs []*Log
	if err := tx.Order("logs.id ASC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	attachLogChannelDisplays(logs)
	for i := range logs {
		if logs[i].Other == "" {
			continue
		}
		otherMap, errParse := common.StrToMap(logs[i].Other)
		if errParse != nil || otherMap == nil {
			continue
		}
		delete(otherMap, "channel_name")
		logs[i].Other = common.MapToJsonStr(otherMap)
	}
	return logs, total, nil
}

// LoadInviterUsernameByUserIDs 批量解析用户对应代理商用户名。
func LoadInviterUsernameByUserIDs(userIDs []int) map[int]string {
	out := make(map[int]string)
	if len(userIDs) == 0 {
		return out
	}
	idSet := make(map[int]struct{}, len(userIDs))
	for _, id := range userIDs {
		if id > 0 {
			idSet[id] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return out
	}
	ids := make([]int, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	var users []User
	if err := DB.Select("id", "inviter_id").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return out
	}
	inviterSet := make(map[int]struct{})
	for _, u := range users {
		if u.InviterId > 0 {
			inviterSet[u.InviterId] = struct{}{}
		}
	}
	if len(inviterSet) == 0 {
		return out
	}
	inviterIDs := make([]int, 0, len(inviterSet))
	for id := range inviterSet {
		inviterIDs = append(inviterIDs, id)
	}
	var inviters []User
	if err := DB.Select("id", "username").Where("id IN ?", inviterIDs).Find(&inviters).Error; err != nil {
		return out
	}
	inviterName := make(map[int]string, len(inviters))
	for _, inv := range inviters {
		inviterName[inv.Id] = inv.Username
	}
	for _, u := range users {
		if name, ok := inviterName[u.InviterId]; ok {
			out[u.Id] = name
		}
	}
	return out
}

// InvoiceRequestAdminItem 运营端发票申请列表项。
type InvoiceRequestAdminItem struct {
	InvoiceRequest
	Username string `json:"username"`
	Email    string `json:"email"`
}

func ListInvoiceRequestsAdminEnriched(status string, pageInfo *common.PageInfo) ([]InvoiceRequestAdminItem, int64, error) {
	rows, total, err := ListInvoiceRequestsAdmin(status, pageInfo)
	if err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return []InvoiceRequestAdminItem{}, total, nil
	}
	userIDs := make([]int, 0, len(rows))
	for _, row := range rows {
		if row != nil && row.UserId > 0 {
			userIDs = append(userIDs, row.UserId)
		}
	}
	userMap := make(map[int]*User)
	if len(userIDs) > 0 {
		var users []User
		if err := DB.Select("id", "username", "email").Where("id IN ?", userIDs).Find(&users).Error; err == nil {
			for i := range users {
				userMap[users[i].Id] = &users[i]
			}
		}
	}
	out := make([]InvoiceRequestAdminItem, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		item := InvoiceRequestAdminItem{InvoiceRequest: *row}
		if u := userMap[row.UserId]; u != nil {
			item.Username = u.Username
			item.Email = u.Email
		}
		out = append(out, item)
	}
	return out, total, nil
}

// BackfillTopUpConsumeAttribution 按用户 used_quota 与成功充值订单 FIFO 回填消耗归因（一次性运维）。
func BackfillTopUpConsumeAttribution(userID int) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}
	var user User
	if err := DB.Select("id", "used_quota").Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&TopUpConsumeAttribution{}).Error; err != nil {
			return err
		}
		if user.UsedQuota <= 0 {
			return nil
		}
		var topups []TopUp
		if err := tx.Where("user_id = ? AND status = ?", userID, common.TopUpStatusSuccess).
			Order("create_time asc").Find(&topups).Error; err != nil {
			return err
		}
		remaining := user.UsedQuota
		for _, topUp := range topups {
			if remaining <= 0 {
				break
			}
			capacity := topUp.ResolveQuotaToAdd()
			if capacity <= 0 {
				continue
			}
			add := remaining
			if add > capacity {
				add = capacity
			}
			attr := TopUpConsumeAttribution{
				UserId:        userID,
				TopUpId:       topUp.Id,
				ConsumedQuota: add,
			}
			if err := tx.Create(&attr).Error; err != nil {
				return err
			}
			remaining -= add
		}
		return nil
	})
}
