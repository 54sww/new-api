package ratio_setting

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

// 用户/用户分组 × 模型 专属计费倍率。
// 命中时作为分组倍率的覆盖值（替换而非叠乘），未命中时回退到
// GroupGroupRatio / GroupRatio 的既有解析逻辑。

const (
	UserModelRatioOptionKey  = "group_ratio_setting.user_model_ratio"
	GroupModelRatioOptionKey = "group_ratio_setting.group_model_ratio"
)

var userModelRatioMap = types.NewRWMap[string, map[string]float64]()  // 用户ID -> 模型匹配模式 -> 倍率
var groupModelRatioMap = types.NewRWMap[string, map[string]float64]() // 用户分组 -> 模型匹配模式 -> 倍率

// ResolveGroupRatio 统一解析最终生效的分组倍率（覆盖语义）：
// 用户×模型专属 > 用户分组×模型专属 > 分组专属分组(GroupGroupRatio) > 普通分组倍率。
// model 为计费模型名，内部先做 FormatMatchingModelName 归一化。
func ResolveGroupRatio(userId int, userGroup, usingGroup, model string) types.GroupRatioInfo {
	info := types.GroupRatioInfo{
		GroupRatio:          1.0,
		GroupSpecialRatio:   -1,
		DedicatedModelRatio: -1,
	}
	matchName := FormatMatchingModelName(model)

	if userId > 0 {
		if patterns, ok := userModelRatioMap.Get(strconv.Itoa(userId)); ok {
			if ratio, pattern, matched := matchModelRatioPattern(patterns, matchName); matched {
				info.GroupRatio = ratio
				info.DedicatedModelRatio = ratio
				info.DedicatedModelSource = fmt.Sprintf("user:%d:%s", userId, pattern)
				return info
			}
		}
	}

	if userGroup != "" {
		if patterns, ok := groupModelRatioMap.Get(userGroup); ok {
			if ratio, pattern, matched := matchModelRatioPattern(patterns, matchName); matched {
				info.GroupRatio = ratio
				info.DedicatedModelRatio = ratio
				info.DedicatedModelSource = fmt.Sprintf("group:%s:%s", userGroup, pattern)
				return info
			}
		}
	}

	if userGroupRatio, ok := GetGroupGroupRatio(userGroup, usingGroup); ok {
		info.GroupSpecialRatio = userGroupRatio
		info.GroupRatio = userGroupRatio
		info.HasSpecialRatio = true
		return info
	}

	info.GroupRatio = GetGroupRatio(usingGroup)
	return info
}

// matchModelRatioPattern 在模式表中解析模型倍率：精确匹配 > 最长前缀通配（pattern 以 * 结尾）> *。
// 非法倍率值（负数/NaN/Inf）视为未命中并告警，防止异常配置影响计费。
func matchModelRatioPattern(patterns map[string]float64, matchName string) (float64, string, bool) {
	if len(patterns) == 0 || matchName == "" {
		return -1, "", false
	}
	if ratio, ok := patterns[matchName]; ok && isValidDedicatedRatio(ratio, matchName) {
		return ratio, matchName, true
	}
	bestLen := -1
	var bestRatio float64
	var bestPattern string
	for pattern, ratio := range patterns {
		var prefix string
		if pattern == "*" {
			prefix = ""
		} else if len(pattern) >= 2 && pattern[len(pattern)-1] == '*' {
			prefix = pattern[:len(pattern)-1]
		} else {
			continue
		}
		if len(prefix) <= bestLen || !strings.HasPrefix(matchName, prefix) {
			continue
		}
		if !isValidDedicatedRatio(ratio, pattern) {
			continue
		}
		bestLen = len(prefix)
		bestRatio = ratio
		bestPattern = pattern
	}
	if bestLen >= 0 {
		return bestRatio, bestPattern, true
	}
	return -1, "", false
}

func isValidDedicatedRatio(ratio float64, pattern string) bool {
	if ratio >= 0 && !math.IsNaN(ratio) && !math.IsInf(ratio, 0) {
		return true
	}
	common.SysError(fmt.Sprintf("invalid dedicated model ratio %v for pattern %s, ignored", ratio, pattern))
	return false
}

func UpdateUserModelRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(userModelRatioMap, jsonStr)
}

func UpdateGroupModelRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(groupModelRatioMap, jsonStr)
}

func UserModelRatio2JSONString() string {
	return userModelRatioMap.MarshalJSONString()
}

func GroupModelRatio2JSONString() string {
	return groupModelRatioMap.MarshalJSONString()
}

// CheckUserModelRatio 校验用户×模型专属倍率配置（option 保存前调用）。
func CheckUserModelRatio(jsonStr string) error {
	m, err := unmarshalDedicatedRatioConfig(jsonStr)
	if err != nil {
		return err
	}
	for userId, patterns := range m {
		id, err := strconv.Atoi(userId)
		if err != nil || id <= 0 {
			return errors.New("user model ratio key must be a positive user id: " + userId)
		}
		if err := checkModelRatioPatterns(userId, patterns); err != nil {
			return err
		}
	}
	return nil
}

// CheckGroupModelRatio 校验用户分组×模型专属倍率配置（option 保存前调用）。
func CheckGroupModelRatio(jsonStr string) error {
	m, err := unmarshalDedicatedRatioConfig(jsonStr)
	if err != nil {
		return err
	}
	for group, patterns := range m {
		if strings.TrimSpace(group) == "" {
			return errors.New("group model ratio key must not be empty")
		}
		if err := checkModelRatioPatterns(group, patterns); err != nil {
			return err
		}
	}
	return nil
}

// unmarshalDedicatedRatioConfig 解析专属倍率配置 JSON，拒绝空值和 "null"（避免清空配置指针后残留旧数据）。
func unmarshalDedicatedRatioConfig(jsonStr string) (map[string]map[string]float64, error) {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" || trimmed == "null" {
		return nil, errors.New("dedicated model ratio config must be a JSON object")
	}
	m := make(map[string]map[string]float64)
	if err := common.Unmarshal([]byte(trimmed), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func checkModelRatioPatterns(owner string, patterns map[string]float64) error {
	for pattern, ratio := range patterns {
		if pattern == "" {
			return errors.New("model pattern must not be empty: " + owner)
		}
		if pattern != "*" {
			star := strings.Index(pattern, "*")
			if star == 0 || (star >= 0 && star != len(pattern)-1) {
				return errors.New("model pattern only supports exact name, 'prefix-*' or '*': " + pattern)
			}
		}
		if ratio < 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
			return fmt.Errorf("dedicated model ratio must be a non-negative number: %s=%v", pattern, ratio)
		}
	}
	return nil
}
