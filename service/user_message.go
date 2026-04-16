package service

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/model"
)

// PublishUserMessage 发布一条站内消息。
// 约束：标题和内容不能为空，且必须至少指定一个接收目标（指定用户或最小角色）。
func PublishUserMessage(msg *model.UserMessage) error {
	if msg == nil {
		return errors.New("message is nil")
	}
	msg.Title = strings.TrimSpace(msg.Title)
	msg.Content = strings.TrimSpace(msg.Content)
	msg.Type = strings.TrimSpace(msg.Type)
	msg.BizType = strings.TrimSpace(msg.BizType)
	if msg.Title == "" || msg.Content == "" {
		return errors.New("title or content is empty")
	}
	if msg.ReceiverUserID <= 0 && msg.ReceiverMinRole <= 0 {
		return errors.New("receiver is empty")
	}
	return model.CreateUserMessage(msg)
}
