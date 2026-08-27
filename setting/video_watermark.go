package setting

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	VideoWatermarkPolicyOff   = "off"
	VideoWatermarkPolicyAll   = "all"
	VideoWatermarkPolicyUsers = "users"
)

// VideoWatermarkPolicy controls forced AIGC watermark injection for video tasks.
// The default is off for backwards compatibility; administrators can select all/users.
var VideoWatermarkPolicy = VideoWatermarkPolicyOff

var videoWatermarkUserIDs = map[int]struct{}{}
var videoWatermarkUserIDsMutex sync.RWMutex

func VideoWatermarkUserIDsToString() string {
	videoWatermarkUserIDsMutex.RLock()
	defer videoWatermarkUserIDsMutex.RUnlock()
	ids := make([]string, 0, len(videoWatermarkUserIDs))
	for id := range videoWatermarkUserIDs {
		ids = append(ids, strconv.Itoa(id))
	}
	sort.Slice(ids, func(i, j int) bool {
		left, _ := strconv.Atoi(ids[i])
		right, _ := strconv.Atoi(ids[j])
		return left < right
	})
	return strings.Join(ids, ",")
}

func parseVideoWatermarkUserIDs(value string) (map[int]struct{}, error) {
	next := make(map[int]struct{})
	for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' ' }) {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || id <= 0 {
			return nil, &videoWatermarkUserIDError{}
		}
		next[id] = struct{}{}
	}
	return next, nil
}

type videoWatermarkUserIDError struct{}

func (*videoWatermarkUserIDError) Error() string {
	return "视频水印指定用户 ID 必须为正整数"
}

func CheckVideoWatermarkUserIDs(value string) error {
	_, err := parseVideoWatermarkUserIDs(value)
	return err
}

func UpdateVideoWatermarkUserIDs(value string) error {
	next, err := parseVideoWatermarkUserIDs(value)
	if err != nil {
		return err
	}
	videoWatermarkUserIDsMutex.Lock()
	videoWatermarkUserIDs = next
	videoWatermarkUserIDsMutex.Unlock()
	return nil
}

func IsVideoWatermarkForcedForUser(userID int) bool {
	switch strings.ToLower(strings.TrimSpace(VideoWatermarkPolicy)) {
	case VideoWatermarkPolicyAll:
		return true
	case VideoWatermarkPolicyUsers:
		videoWatermarkUserIDsMutex.RLock()
		defer videoWatermarkUserIDsMutex.RUnlock()
		_, ok := videoWatermarkUserIDs[userID]
		return ok
	default:
		return false
	}
}

func SetVideoWatermarkPolicy(value string) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value != VideoWatermarkPolicyAll && value != VideoWatermarkPolicyUsers {
		value = VideoWatermarkPolicyOff
	}
	VideoWatermarkPolicy = value
}

func CheckVideoWatermarkPolicy(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case VideoWatermarkPolicyOff, VideoWatermarkPolicyAll, VideoWatermarkPolicyUsers:
		return nil
	default:
		return &videoWatermarkPolicyError{}
	}
}

type videoWatermarkPolicyError struct{}

func (*videoWatermarkPolicyError) Error() string {
	return "视频水印策略必须为 off、all 或 users"
}
