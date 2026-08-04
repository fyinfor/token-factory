package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// UpdateTaskBulk 薄入口，实际轮询逻辑在 service 层
func UpdateTaskBulk() {
	service.TaskPollingLoop()
}

// UpdateAsrTaskBulk 阿里云 ASR 异步转写后台轮询入口（对齐视频任务轮询）。
func UpdateAsrTaskBulk() {
	relay.AsrTaskPollingLoop()
}

func GetAllTask(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	queryParams, empty := buildSyncTaskQueryParams(c, true)
	if empty {
		pageInfo.SetTotal(0)
		pageInfo.SetItems([]*dto.TaskDto{})
		common.ApiSuccess(c, pageInfo)
		return
	}

	items := model.TaskGetAllTasks(pageInfo.GetStartIdx(), pageInfo.GetPageSize(), queryParams)
	total := model.TaskCountAllTasks(queryParams)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tasksToDto(items, true))
	common.ApiSuccess(c, pageInfo)
}

func GetUserTask(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	userId := c.GetInt("id")

	queryParams, empty := buildSyncTaskQueryParams(c, false)
	if empty {
		pageInfo.SetTotal(0)
		pageInfo.SetItems([]*dto.TaskDto{})
		common.ApiSuccess(c, pageInfo)
		return
	}

	items := model.TaskGetAllUserTask(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), queryParams)
	total := model.TaskCountAllUserTask(userId, queryParams)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tasksToDto(items, false))
	common.ApiSuccess(c, pageInfo)
}

func buildSyncTaskQueryParams(c *gin.Context, isAdmin bool) (model.SyncTaskQueryParams, bool) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	queryParams := model.SyncTaskQueryParams{
		Platform:        constant.TaskPlatform(c.Query("platform")),
		TaskID:          c.Query("task_id"),
		ModelName:       c.Query("model_name"),
		Status:          normalizeTaskStatusQuery(c.Query("status")),
		Action:          c.Query("action"),
		VideoType:       c.Query("video_type"),
		StartTimestamp:  startTimestamp,
		EndTimestamp:    endTimestamp,
		VideoFailedOnly: parseVideoFailedQuery(c.Query("video_failed")),
	}
	if isAdmin {
		queryParams.ChannelID = c.Query("channel_id")
	}

	if isAdmin {
		username := strings.TrimSpace(c.Query("username"))
		if username != "" {
			userID, err := model.GetUserIdByUsername(username)
			if err != nil || userID == 0 {
				return queryParams, true
			}
			queryParams.UserID = strconv.Itoa(userID)
		}
	}

	requestID := strings.TrimSpace(c.Query("request_id"))
	if requestID != "" {
		taskIDs, err := model.ResolveTaskIDsByRequestID(requestID)
		if err != nil || len(taskIDs) == 0 {
			return queryParams, true
		}
		if queryParams.TaskID != "" {
			matched := false
			for _, id := range taskIDs {
				if id == queryParams.TaskID {
					matched = true
					break
				}
			}
			if !matched {
				return queryParams, true
			}
		} else {
			queryParams.TaskIDs = taskIDs
		}
	}

	return queryParams, false
}

func tasksToDto(tasks []*model.Task, fillUser bool) []*dto.TaskDto {
	var userIdMap map[int]*model.UserBase
	if fillUser {
		userIdMap = make(map[int]*model.UserBase)
		userIds := types.NewSet[int]()
		for _, task := range tasks {
			userIds.Add(task.UserId)
		}
		for _, userId := range userIds.Items() {
			cacheUser, err := model.GetUserCache(userId)
			if err == nil {
				userIdMap[userId] = cacheUser
			}
		}
	}
	result := make([]*dto.TaskDto, len(tasks))
	for i, task := range tasks {
		if fillUser {
			if user, ok := userIdMap[task.UserId]; ok {
				task.Username = user.Username
			}
		}
		result[i] = relay.TaskModel2Dto(task)
	}
	return result
}

func parseVideoFailedQuery(raw string) bool {
	raw = strings.TrimSpace(raw)
	return raw == "1" || strings.EqualFold(raw, "true") || raw == "yes"
}

func normalizeTaskStatusQuery(raw string) string {
	raw = strings.TrimSpace(raw)
	switch strings.ToUpper(raw) {
	case "SUCCESS", "成功":
		return string(model.TaskStatusSuccess)
	case "FAILURE", "FAILED", "FAIL", "失败":
		return string(model.TaskStatusFailure)
	default:
		return raw
	}
}
