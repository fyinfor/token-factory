package service

import (
	"fmt"
	"log"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	pb "github.com/QuantumNous/new-api/proto/route"
)

// ExportUsersForTFSync 拉取本站可同步用户（脱敏，不含密码等敏感字段）。
func ExportUsersForTFSync() ([]model.User, error) {
	var users []model.User
	if err := model.DB.Select("id, username, display_name, role, status").
		Order("id ASC").
		Find(&users).Error; err != nil {
		log.Printf("[SYS] ExportUsersForTFSync: query failed: %v", err)
		return nil, err
	}
	log.Printf("[SYS] ExportUsersForTFSync: exported=%d", len(users))
	return users, nil
}

// BuildExternalUserInfosForTF 将本地用户转为 TokenFactory gRPC ExternalUserInfo。
func BuildExternalUserInfosForTF(users []model.User) []*pb.ExternalUserInfo {
	infos := make([]*pb.ExternalUserInfo, 0, len(users))
	for _, u := range users {
		displayName := u.DisplayName
		if displayName == "" {
			displayName = u.Username
		}
		infos = append(infos, &pb.ExternalUserInfo{
			Uid:         int32(u.Id),
			Username:    u.Username,
			DisplayName: displayName,
			Role:        int32(u.Role),
			Status:      int32(u.Status),
		})
	}
	return infos
}

// SyncUsersToTokenFactory 将本站用户快照推送到 TokenFactory（按 TOKENFACTORY_SITE_KEY 隔离）。
// TokenFactory 端仅 upsert external_users，不修改用户路由策略配置。
func SyncUsersToTokenFactory() (pushed int, total int, err error) {
	if !common.TokenFactoryRouteEnabled() {
		return 0, 0, fmt.Errorf("TokenFactory route not enabled")
	}

	users, err := ExportUsersForTFSync()
	if err != nil {
		return 0, 0, err
	}
	infos := BuildExternalUserInfosForTF(users)
	if len(infos) == 0 {
		return 0, 0, nil
	}

	jwt, err := common.IssueTokenFactoryJWT(1, 100)
	if err != nil {
		return 0, len(infos), fmt.Errorf("issue JWT: %w", err)
	}

	count, err := SyncUsersToTF(jwt, infos)
	if err != nil {
		return count, len(infos), err
	}
	return count, len(infos), nil
}
