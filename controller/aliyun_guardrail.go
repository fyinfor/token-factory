package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetAliyunGuardrailLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	logs, total, err := model.GetAliyunGuardrailLogs(
		pageInfo.GetStartIdx(),
		pageInfo.GetPageSize(),
		model.AliyunGuardrailLogFilter{
			Username:       c.Query("username"),
			StartTimestamp: startTimestamp,
			EndTimestamp:   endTimestamp,
		},
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
}
