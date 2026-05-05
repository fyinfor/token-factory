package setting

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

var RateLimitUserWhitelist = map[int]struct{}{}
var RateLimitUserWhitelistMutex sync.RWMutex

func RateLimitUserWhitelist2JSONString() string {
	RateLimitUserWhitelistMutex.RLock()
	defer RateLimitUserWhitelistMutex.RUnlock()

	ids := make([]int, 0, len(RateLimitUserWhitelist))
	for id := range RateLimitUserWhitelist {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	jsonBytes, err := json.Marshal(ids)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

func UpdateRateLimitUserWhitelistByJSONString(jsonStr string) error {
	var ids []int
	if err := json.Unmarshal([]byte(jsonStr), &ids); err != nil {
		return err
	}

	next := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return fmt.Errorf("invalid user id in whitelist: %d", id)
		}
		next[id] = struct{}{}
	}

	RateLimitUserWhitelistMutex.Lock()
	defer RateLimitUserWhitelistMutex.Unlock()
	RateLimitUserWhitelist = next
	return nil
}

func CheckRateLimitUserWhitelistJSON(jsonStr string) error {
	var ids []int
	if err := json.Unmarshal([]byte(jsonStr), &ids); err != nil {
		return err
	}
	for _, id := range ids {
		if id <= 0 {
			return fmt.Errorf("invalid user id in whitelist: %d", id)
		}
	}
	return nil
}

func IsUserInRateLimitWhitelist(userId int) bool {
	if userId <= 0 {
		return false
	}
	RateLimitUserWhitelistMutex.RLock()
	defer RateLimitUserWhitelistMutex.RUnlock()
	_, ok := RateLimitUserWhitelist[userId]
	return ok
}
