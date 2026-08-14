package setting

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const AliyunGuardrailBlockedReply = `抱歉，我无法协助处理这个请求。你可以换个安全、合规的方向提问，我会尽力帮助。`

var AliyunGuardrailEnabled = false
var AliyunGuardrailInputEnabled = true
var AliyunGuardrailOutputEnabled = true
var AliyunGuardrailVideoEnabled = false
var AliyunGuardrailHidePlaygroundMediaTabs = false
var AliyunGuardrailAccessKeyID = ``
var AliyunGuardrailAccessKeySecret = ``
var AliyunGuardrailRegionID = `cn-shanghai`

var aliyunGuardrailUserIDs = map[int]struct{}{}
var aliyunGuardrailUserIDsMutex sync.RWMutex

func AliyunGuardrailConfigured() bool {
	return strings.TrimSpace(AliyunGuardrailAccessKeyID) != `` && strings.TrimSpace(AliyunGuardrailAccessKeySecret) != ``
}

func ShouldCheckAliyunGuardrailInput() bool {
	return AliyunGuardrailEnabled && AliyunGuardrailInputEnabled && AliyunGuardrailConfigured()
}

func ShouldCheckAliyunGuardrailOutput() bool {
	return AliyunGuardrailEnabled && AliyunGuardrailOutputEnabled && AliyunGuardrailConfigured()
}

func ShouldCheckAliyunGuardrailVideo() bool {
	return ShouldCheckAliyunGuardrailOutput() && AliyunGuardrailVideoEnabled
}

// AliyunGuardrailUserIDsToString returns the configured user scope as comma-separated user IDs.
// An empty value means the guardrail applies to all users.
func AliyunGuardrailUserIDsToString() string {
	aliyunGuardrailUserIDsMutex.RLock()
	defer aliyunGuardrailUserIDsMutex.RUnlock()

	ids := make([]int, 0, len(aliyunGuardrailUserIDs))
	for id := range aliyunGuardrailUserIDs {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, strconv.Itoa(id))
	}
	return strings.Join(values, ",")
}

func parseAliyunGuardrailUserIDs(value string) (map[int]struct{}, error) {
	next := make(map[int]struct{})
	for lineNumber, line := range strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n") {
		for _, part := range strings.Split(line, ",") {
			idValue := strings.TrimSpace(part)
			if idValue == "" {
				continue
			}
			id, err := strconv.Atoi(idValue)
			if err != nil || id <= 0 {
				return nil, fmt.Errorf("第 %d 行的用户 ID 必须为正整数", lineNumber+1)
			}
			next[id] = struct{}{}
		}
	}
	return next, nil
}

func CheckAliyunGuardrailUserIDs(value string) error {
	_, err := parseAliyunGuardrailUserIDs(value)
	return err
}

func UpdateAliyunGuardrailUserIDs(value string) error {
	next, err := parseAliyunGuardrailUserIDs(value)
	if err != nil {
		return err
	}
	aliyunGuardrailUserIDsMutex.Lock()
	aliyunGuardrailUserIDs = next
	aliyunGuardrailUserIDsMutex.Unlock()
	return nil
}

func HasAliyunGuardrailUserScope() bool {
	aliyunGuardrailUserIDsMutex.RLock()
	defer aliyunGuardrailUserIDsMutex.RUnlock()
	return len(aliyunGuardrailUserIDs) > 0
}

func IsAliyunGuardrailEnabledForUser(userID int) bool {
	aliyunGuardrailUserIDsMutex.RLock()
	defer aliyunGuardrailUserIDsMutex.RUnlock()
	if len(aliyunGuardrailUserIDs) == 0 {
		return true
	}
	_, ok := aliyunGuardrailUserIDs[userID]
	return ok
}

func ShouldCheckAliyunGuardrailInputForUser(userID int) bool {
	return ShouldCheckAliyunGuardrailInput() && IsAliyunGuardrailEnabledForUser(userID)
}

func ShouldCheckAliyunGuardrailOutputForUser(userID int) bool {
	return ShouldCheckAliyunGuardrailOutput() && IsAliyunGuardrailEnabledForUser(userID)
}

func ShouldCheckAliyunGuardrailVideoForUser(userID int) bool {
	return ShouldCheckAliyunGuardrailVideo() && IsAliyunGuardrailEnabledForUser(userID)
}
