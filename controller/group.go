package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func GetGroups(c *gin.Context) {
	// 已审核供应商仅返回其自有渠道里出现过的分组；管理员保持全量返回。
	if c.GetInt("role") < common.RoleAdminUser {
		ownerUserID := c.GetInt("id")
		channels, _, err := model.SearchSupplierChannels(&ownerUserID, 0, 100000, model.SupplierChannelSearchFilter{})
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		groupSet := make(map[string]struct{})
		for _, channel := range channels {
			for _, groupName := range channel.GetGroups() {
				groupName = strings.TrimSpace(groupName)
				if groupName == "" {
					continue
				}
				groupSet[groupName] = struct{}{}
			}
		}
		groupNames := make([]string, 0, len(groupSet))
		for groupName := range groupSet {
			groupNames = append(groupNames, groupName)
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data":    groupNames,
		})
		return
	}

	groupNames := make([]string, 0)
	for groupName := range ratio_setting.GetGroupRatioCopy() {
		groupNames = append(groupNames, groupName)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    groupNames,
	})
}

func GetUserGroups(c *gin.Context) {
	usableGroups := make(map[string]map[string]interface{})
	userGroup := ""
	userId := c.GetInt("id")
	userGroup, _ = model.GetUserGroup(userId, false)
	userUsableGroups := service.GetUserUsableGroups(userGroup)
	for groupName, _ := range ratio_setting.GetGroupRatioCopy() {
		// UserUsableGroups contains the groups that the user can use
		if desc, ok := userUsableGroups[groupName]; ok {
			usableGroups[groupName] = map[string]interface{}{
				"ratio": service.GetUserGroupRatio(userGroup, groupName),
				"desc":  desc,
			}
		}
	}
	if _, ok := userUsableGroups["auto"]; ok {
		usableGroups["auto"] = map[string]interface{}{
			"ratio": "自动",
			"desc":  setting.GetUsableGroupDescription("auto"),
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    usableGroups,
	})
}
