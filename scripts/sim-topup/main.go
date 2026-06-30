// 本地开发：模拟用户充值并触发分销提成（直连数据库，无需登录）。
// 用法: go run ./scripts/sim-topup <用户名> <美元金额>
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load(".env")
	common.InitEnv()
	_ = common.InitRedisClient()

	if err := model.InitDB(); err != nil {
		fmt.Fprintf(os.Stderr, "数据库迁移警告(可忽略): %v\n", err)
	}
	if err := model.InitLogDB(); err != nil {
		fmt.Fprintf(os.Stderr, "日志库初始化失败: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = model.CloseDB() }()

	if len(os.Args) < 3 {
		fmt.Println("用法: topup <用户名> <充值美元>")
		fmt.Println("示例: topup test1 10")
		os.Exit(1)
	}

	username := strings.TrimSpace(os.Args[1])
	usd, err := strconv.ParseFloat(strings.TrimSpace(os.Args[2]), 64)
	if err != nil || usd <= 0 {
		fmt.Fprintf(os.Stderr, "充值金额无效，请输入大于 0 的数字（美元）\n")
		os.Exit(1)
	}

	var user model.User
	if err := model.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			fmt.Fprintf(os.Stderr, "用户不存在: %s\n", username)
		} else {
			fmt.Fprintf(os.Stderr, "查询用户失败: %v\n", err)
		}
		os.Exit(1)
	}

	quota := common.QuotaFromUSD(usd)
	if quota <= 0 {
		fmt.Fprintf(os.Stderr, "换算额度为 0\n")
		os.Exit(1)
	}

	if err := model.IncreaseUserQuota(user.Id, quota, true); err != nil {
		fmt.Fprintf(os.Stderr, "入账失败: %v\n", err)
		os.Exit(1)
	}
	model.RecordLog(user.Id, model.LogTypeTopup,
		fmt.Sprintf("[模拟充值] %s", logger.LogQuota(quota)))
	model.ApplyAffiliateTopupReward(user.Id, quota)

	fmt.Printf("已为 %s (id=%d) 充值 %s（约 %.2f USD）\n",
		user.Username, user.Id, logger.LogQuota(quota), usd)

	if user.InviterId <= 0 {
		fmt.Println("该用户无邀请人，不会产生分销收益。")
		return
	}

	inviter, err := model.GetUserById(user.InviterId, false)
	if err != nil {
		fmt.Printf("邀请人 id=%d 查询失败: %v\n", user.InviterId, err)
		return
	}
	if !model.UserIsDistributor(inviter) {
		fmt.Printf("邀请人 %s 不是分销商，无分销收益。\n", inviter.Username)
		return
	}

	inviterAfter, _ := model.GetUserById(inviter.Id, false)
	fmt.Printf("分销商 %s (id=%d) 待使用收益 aff_quota: %s\n",
		inviterAfter.Username, inviterAfter.Id, logger.LogQuota(inviterAfter.AffQuota))
	fmt.Println("刷新: http://localhost:5173/console/distributor/center")
}
