package ratio_setting

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withDedicatedRatioConfig 临时替换全局倍率表，测试结束后恢复原状。
func withDedicatedRatioConfig(t *testing.T, userModelJson, groupModelJson, groupGroupJson, groupJson string) {
	t.Helper()
	origUserModel := UserModelRatio2JSONString()
	origGroupModel := GroupModelRatio2JSONString()
	origGroupGroup := GroupGroupRatio2JSONString()
	origGroup := GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateUserModelRatioByJSONString(origUserModel))
		require.NoError(t, UpdateGroupModelRatioByJSONString(origGroupModel))
		require.NoError(t, UpdateGroupGroupRatioByJSONString(origGroupGroup))
		require.NoError(t, UpdateGroupRatioByJSONString(origGroup))
	})
	require.NoError(t, UpdateUserModelRatioByJSONString(userModelJson))
	require.NoError(t, UpdateGroupModelRatioByJSONString(groupModelJson))
	require.NoError(t, UpdateGroupGroupRatioByJSONString(groupGroupJson))
	require.NoError(t, UpdateGroupRatioByJSONString(groupJson))
}

func TestMatchModelRatioPattern(t *testing.T) {
	tests := []struct {
		name          string
		patterns      map[string]float64
		model         string
		expectRatio   float64
		expectPattern string
		expectMatched bool
	}{
		{
			name:          "exact beats wildcard and star",
			patterns:      map[string]float64{"gpt-4": 0.8, "gpt-*": 0.5, "*": 1.0},
			model:         "gpt-4",
			expectRatio:   0.8,
			expectPattern: "gpt-4",
			expectMatched: true,
		},
		{
			name:          "longest prefix wins",
			patterns:      map[string]float64{"deep-*": 0.7, "deepseek-*": 0.75, "*": 1.0},
			model:         "deepseek-chat",
			expectRatio:   0.75,
			expectPattern: "deepseek-*",
			expectMatched: true,
		},
		{
			name:          "star fallback",
			patterns:      map[string]float64{"gpt-4": 0.8, "*": 1.0},
			model:         "claude-3",
			expectRatio:   1.0,
			expectPattern: "*",
			expectMatched: true,
		},
		{
			name:          "no match",
			patterns:      map[string]float64{"gpt-*": 0.5},
			model:         "claude-3",
			expectRatio:   -1,
			expectPattern: "",
			expectMatched: false,
		},
		{
			name:          "empty patterns",
			patterns:      map[string]float64{},
			model:         "gpt-4",
			expectRatio:   -1,
			expectPattern: "",
			expectMatched: false,
		},
		{
			name:          "empty model",
			patterns:      map[string]float64{"*": 1.0},
			model:         "",
			expectRatio:   -1,
			expectPattern: "",
			expectMatched: false,
		},
		{
			name:          "invalid exact ratio falls back to wildcard",
			patterns:      map[string]float64{"gpt-4": -1, "gpt-*": 0.5},
			model:         "gpt-4",
			expectRatio:   0.5,
			expectPattern: "gpt-*",
			expectMatched: true,
		},
		{
			name:          "NaN ratio ignored",
			patterns:      map[string]float64{"gpt-4": math.NaN()},
			model:         "gpt-4",
			expectRatio:   -1,
			expectPattern: "",
			expectMatched: false,
		},
		{
			name:          "pattern without star only matches exactly",
			patterns:      map[string]float64{"gpt": 0.5},
			model:         "gpt-4",
			expectRatio:   -1,
			expectPattern: "",
			expectMatched: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio, pattern, matched := matchModelRatioPattern(tt.patterns, tt.model)
			assert.Equal(t, tt.expectMatched, matched)
			assert.Equal(t, tt.expectRatio, ratio)
			assert.Equal(t, tt.expectPattern, pattern)
		})
	}
}

func TestResolveGroupRatioPriority(t *testing.T) {
	withDedicatedRatioConfig(t,
		`{"42":{"glm-5.1":0.6}}`,
		`{"vip":{"glm-5.1":0.9}}`,
		`{"vip":{"default":0.8}}`,
		`{"default":2,"vip":1}`,
	)

	t.Run("user model rule wins over everything", func(t *testing.T) {
		info := ResolveGroupRatio(42, "vip", "default", "glm-5.1")
		assert.Equal(t, 0.6, info.GroupRatio)
		assert.Equal(t, 0.6, info.DedicatedModelRatio)
		assert.Equal(t, "user:42:glm-5.1", info.DedicatedModelSource)
		assert.False(t, info.HasSpecialRatio)
		assert.Equal(t, -1.0, info.GroupSpecialRatio)
	})

	t.Run("group model rule wins over group group ratio", func(t *testing.T) {
		info := ResolveGroupRatio(43, "vip", "default", "glm-5.1")
		assert.Equal(t, 0.9, info.GroupRatio)
		assert.Equal(t, "group:vip:glm-5.1", info.DedicatedModelSource)
		assert.False(t, info.HasSpecialRatio)
	})

	t.Run("group group ratio wins over plain group ratio", func(t *testing.T) {
		info := ResolveGroupRatio(43, "vip", "default", "gpt-4o")
		assert.Equal(t, 0.8, info.GroupRatio)
		assert.Equal(t, 0.8, info.GroupSpecialRatio)
		assert.True(t, info.HasSpecialRatio)
		assert.Empty(t, info.DedicatedModelSource)
		assert.Equal(t, -1.0, info.DedicatedModelRatio)
	})

	t.Run("falls back to plain group ratio", func(t *testing.T) {
		info := ResolveGroupRatio(43, "default", "default", "gpt-4o")
		assert.Equal(t, 2.0, info.GroupRatio)
		assert.False(t, info.HasSpecialRatio)
		assert.Empty(t, info.DedicatedModelSource)
	})

	t.Run("unknown group defaults to 1", func(t *testing.T) {
		info := ResolveGroupRatio(43, "default", "unknown", "gpt-4o")
		assert.Equal(t, 1.0, info.GroupRatio)
	})

	t.Run("user rule not consulted for other users", func(t *testing.T) {
		info := ResolveGroupRatio(999, "default", "default", "glm-5.1")
		assert.Equal(t, 2.0, info.GroupRatio)
		assert.Empty(t, info.DedicatedModelSource)
	})

	t.Run("userId zero skips user rules", func(t *testing.T) {
		info := ResolveGroupRatio(0, "vip", "default", "glm-5.1")
		assert.Equal(t, 0.9, info.GroupRatio)
		assert.Equal(t, "group:vip:glm-5.1", info.DedicatedModelSource)
	})
}

func TestResolveGroupRatioModelNormalization(t *testing.T) {
	withDedicatedRatioConfig(t,
		`{"1":{"gpt-4-gizmo-*":0.7,"deepseek-*":0.75}}`,
		`{}`, `{"default":{}}`, `{"default":1}`,
	)

	info := ResolveGroupRatio(1, "default", "default", "gpt-4-gizmo-abc123")
	assert.Equal(t, 0.7, info.GroupRatio)
	assert.Equal(t, "user:1:gpt-4-gizmo-*", info.DedicatedModelSource)

	info = ResolveGroupRatio(1, "default", "default", "deepseek-chat")
	assert.Equal(t, 0.75, info.GroupRatio)
}

func TestResolveGroupRatioEmptyConfigBackwardCompatible(t *testing.T) {
	withDedicatedRatioConfig(t, `{}`, `{}`, `{"vip":{"default":0.8}}`, `{"default":2,"vip":1}`)

	info := ResolveGroupRatio(42, "vip", "default", "gpt-4o")
	assert.Equal(t, 0.8, info.GroupRatio)
	assert.True(t, info.HasSpecialRatio)
	assert.Empty(t, info.DedicatedModelSource)

	info = ResolveGroupRatio(42, "default", "default", "gpt-4o")
	assert.Equal(t, 2.0, info.GroupRatio)
	assert.Empty(t, info.DedicatedModelSource)
}

func TestCheckUserModelRatio(t *testing.T) {
	require.NoError(t, CheckUserModelRatio(`{}`))
	require.NoError(t, CheckUserModelRatio(`{"42":{"glm-5.1":0.65,"deepseek-*":0.7,"*":1}}`))

	assert.Error(t, CheckUserModelRatio(`not json`))
	assert.Error(t, CheckUserModelRatio(``))
	assert.Error(t, CheckUserModelRatio(`null`))
	assert.Error(t, CheckUserModelRatio(`{"abc":{"glm":1}}`))
	assert.Error(t, CheckUserModelRatio(`{"0":{"glm":1}}`))
	assert.Error(t, CheckUserModelRatio(`{"-1":{"glm":1}}`))
	assert.Error(t, CheckUserModelRatio(`{"42":{"glm":-1}}`))
	assert.Error(t, CheckUserModelRatio(`{"42":{"":1}}`))
	assert.Error(t, CheckUserModelRatio(`{"42":{"*glm":1}}`))
	assert.Error(t, CheckUserModelRatio(`{"42":{"gl*m":1}}`))
}

func TestCheckGroupModelRatio(t *testing.T) {
	require.NoError(t, CheckGroupModelRatio(`{}`))
	require.NoError(t, CheckGroupModelRatio(`{"vip":{"glm-5.1":0.85,"gpt-*":0.8}}`))

	assert.Error(t, CheckGroupModelRatio(`not json`))
	assert.Error(t, CheckGroupModelRatio(``))
	assert.Error(t, CheckGroupModelRatio(`null`))
	assert.Error(t, CheckGroupModelRatio(`{"":{"glm":1}}`))
	assert.Error(t, CheckGroupModelRatio(`{"vip":{"glm":-0.1}}`))
	assert.Error(t, CheckGroupModelRatio(`{"vip":{"gl*m":1}}`))
}
