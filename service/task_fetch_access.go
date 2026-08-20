package service

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// LoadTaskForAsyncFetch 加载视频/Suno 等异步任务供查询：
// 管理员可跨用户读取；普通用户仍按归属过滤；命中后再用令牌模型白名单校验。
func LoadTaskForAsyncFetch(c *gin.Context, taskID string) (*model.Task, *dto.TaskError) {
	ownerUserID := FetchOwnerUserID(c)
	var (
		task  *model.Task
		exist bool
		err   error
	)
	if ownerUserID == 0 {
		task, exist, err = model.GetByOnlyTaskId(taskID)
	} else {
		task, exist, err = model.GetByTaskId(ownerUserID, taskID)
	}
	if err != nil {
		return nil, TaskErrorWrapper(err, "get_task_failed", http.StatusInternalServerError)
	}
	if !exist || task == nil {
		return nil, TaskErrorWrapperLocal(errors.New("task_not_exist"), "task_not_exist", http.StatusBadRequest)
	}
	if fetchErr := denyIfTokenModelForbidden(c, task.GetFetchAccessModelName()); fetchErr != nil {
		return nil, fetchErr
	}
	return task, nil
}

// LoadTasksForAsyncFetch 批量加载异步任务（Suno fetch）。管理员不限制归属；返回结果再按模型白名单过滤。
func LoadTasksForAsyncFetch(c *gin.Context, taskIDs []any) ([]*model.Task, *dto.TaskError) {
	if len(taskIDs) == 0 {
		return []*model.Task{}, nil
	}
	ownerUserID := FetchOwnerUserID(c)
	var (
		tasks []*model.Task
		err   error
	)
	if ownerUserID == 0 {
		tasks, err = model.GetByTaskIdsOnly(taskIDs)
	} else {
		tasks, err = model.GetByTaskIds(ownerUserID, taskIDs)
	}
	if err != nil {
		return nil, TaskErrorWrapper(err, "get_tasks_failed", http.StatusInternalServerError)
	}
	allowed := make([]*model.Task, 0, len(tasks))
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if !TokenAllowsModel(c, task.GetFetchAccessModelName()) {
			continue
		}
		allowed = append(allowed, task)
	}
	return allowed, nil
}

// LoadAsrTaskForAsyncFetch 加载 ASR 异步任务：管理员可跨用户读取，随后校验令牌模型白名单。
func LoadAsrTaskForAsyncFetch(c *gin.Context, taskID string) (*model.AsrTask, *types.TokenFactoryError) {
	task, err := model.GetAsrTaskByTaskID(taskID, FetchOwnerUserID(c))
	if err != nil {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("任务不存在: %s", taskID),
			types.ErrorCodeInvalidRequest, http.StatusNotFound, types.ErrOptionWithSkipRetry())
	}
	if !TokenAllowsModel(c, task.Model) {
		return nil, types.NewErrorWithStatusCode(
			errors.New(TokenModelForbiddenMessage(c, task.Model)),
			types.ErrorCodeAccessDenied, http.StatusForbidden, types.ErrOptionWithSkipRetry())
	}
	return task, nil
}

func denyIfTokenModelForbidden(c *gin.Context, modelName string) *dto.TaskError {
	if TokenAllowsModel(c, modelName) {
		return nil
	}
	return TaskErrorWrapperLocal(
		errors.New(TokenModelForbiddenMessage(c, modelName)),
		"token_model_forbidden",
		http.StatusForbidden,
	)
}
