package tencentvod

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const defaultVODRegion = "ap-guangzhou"

type Credentials struct {
	SubAppID  uint64
	SecretID  string
	SecretKey string
	Region    string
}

func ParseCredentials(raw string) (Credentials, error) {
	s := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "Bearer "))
	if s == "" {
		return Credentials{}, errors.New("empty tencent cloud vod credentials")
	}

	var parts []string
	if strings.Contains(s, "|") {
		for _, p := range strings.Split(s, "|") {
			p = strings.TrimSpace(p)
			if p != "" {
				parts = append(parts, p)
			}
		}
	} else {
		for _, line := range strings.Split(s, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				parts = append(parts, line)
			}
		}
	}

	if len(parts) < 3 {
		return Credentials{}, fmt.Errorf("invalid credentials: need SubAppId, SecretId and SecretKey (%d segments)", len(parts))
	}
	subID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || subID == 0 {
		return Credentials{}, fmt.Errorf("invalid SubAppId %q", parts[0])
	}

	cred := Credentials{
		SubAppID:  subID,
		SecretID:  parts[1],
		SecretKey: parts[2],
		Region:    defaultVODRegion,
	}
	if len(parts) >= 4 && strings.TrimSpace(parts[3]) != "" {
		cred.Region = strings.TrimSpace(parts[3])
	}
	return cred, nil
}

func SplitCombinedModel(combined string) (modelName, modelVersion string) {
	combined = strings.TrimSpace(combined)
	if combined == "" {
		return "", ""
	}
	idx := strings.Index(combined, "-")
	if idx <= 0 || idx >= len(combined)-1 {
		return combined, ""
	}
	return strings.TrimSpace(combined[:idx]), strings.TrimSpace(combined[idx+1:])
}

package tencentvod

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const defaultVODRegion = "ap-guangzhou"

type Credentials struct {
	SubAppID  uint64
	SecretID  string
	SecretKey string
	Region    string
}

func ParseCredentials(raw string) (Credentials, error) {
	s := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "Bearer "))
	if s == "" {
		return Credentials{}, errors.New("empty tencent cloud vod credentials")
	}

	var parts []string
	if strings.Contains(s, "|") {
		for _, p := range strings.Split(s, "|") {
			p = strings.TrimSpace(p)
			if p != "" {
				parts = append(parts, p)
			}
		}
	} else {
		for _, line := range strings.Split(s, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				parts = append(parts, line)
			}
		}
	}

	if len(parts) < 3 {
		return Credentials{}, fmt.Errorf("invalid credentials: need SubAppId, SecretId and SecretKey (%d segments)", len(parts))
	}

	subID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || subID == 0 {
		return Credentials{}, fmt.Errorf("invalid SubAppId %q", parts[0])
	}

	c := Credentials{
		SubAppID:  subID,
		SecretID:  parts[1],
		SecretKey: parts[2],
		Region:    defaultVODRegion,
	}
	if len(parts) >= 4 && strings.TrimSpace(parts[3]) != "" {
		c.Region = strings.TrimSpace(parts[3])
	}
	return c, nil
}

func SplitCombinedModel(combined string) (modelName, modelVersion string) {
	combined = strings.TrimSpace(combined)
	if combined == "" {
		return "", ""
	}
	idx := strings.Index(combined, "-")
	if idx <= 0 || idx >= len(combined)-1 {
		return combined, ""
	}
	return strings.TrimSpace(combined[:idx]), strings.TrimSpace(combined[idx+1:])
}

