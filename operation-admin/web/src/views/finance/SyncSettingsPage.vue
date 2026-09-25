<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getBalanceSyncSettingsApi,
  putBalanceSyncSettingsApi,
  syncBalancesApi,
  type BalanceSyncSettings,
} from '@/api/finance'

const loading = ref(false)
const saving = ref(false)
const data = ref<BalanceSyncSettings | null>(null)
const form = reactive({
  enabled: true,
  interval_minutes: 5,
})

function formatTime(ts?: number) {
  if (!ts) return '—'
  return new Date(ts * 1000).toLocaleString()
}

async function load() {
  loading.value = true
  try {
    const res = await getBalanceSyncSettingsApi()
    data.value = res
    form.enabled = !!res.enabled
    form.interval_minutes = res.interval_minutes || 5
  } finally {
    loading.value = false
  }
}

async function onSave() {
  if (form.interval_minutes < 1 || form.interval_minutes > 1440) {
    ElMessage.warning('间隔需在 1～1440 分钟之间')
    return
  }
  saving.value = true
  try {
    const res = await putBalanceSyncSettingsApi({
      enabled: form.enabled,
      interval_minutes: form.interval_minutes,
    })
    data.value = { ...data.value, ...res }
    ElMessage.success('已保存，将按新间隔生效')
    await load()
  } finally {
    saving.value = false
  }
}

async function onRunNow() {
  try {
    await syncBalancesApi()
    ElMessage.success('已触发一次余额同步')
    await load()
  } catch {
    // ignore
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="page-container" v-loading="loading">
    <div class="page-header mb-16">
      <div>
        <h2 class="page-title">自动同步</h2>
        <p class="page-hint">
          按分钟间隔拉取已开启「同步余额」的渠道余额，写入快照并判断是否通知；日报据此统计当日消耗。
        </p>
      </div>
      <div class="page-actions">
        <el-button @click="load">刷新</el-button>
        <el-button v-permission="'finance:reconcile:sync'" @click="onRunNow">立即同步一次</el-button>
        <el-button v-permission="'finance:sync-settings:edit'" type="primary" :loading="saving" @click="onSave">
          保存
        </el-button>
      </div>
    </div>

    <el-card shadow="never" class="mb-16">
      <el-form label-width="140px" style="max-width: 520px">
        <el-form-item label="启用自动同步">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item label="同步间隔（分钟）">
          <el-input-number v-model="form.interval_minutes" :min="1" :max="1440" :step="1" />
          <span class="form-hint">最小 1 分钟，最大 1440（1 天）</span>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <template #header>最近一次自动同步</template>
      <el-descriptions :column="1" border size="small">
        <el-descriptions-item label="状态">{{ data?.last_sync_message || '—' }}</el-descriptions-item>
        <el-descriptions-item label="成功渠道数">{{ data?.last_sync_count ?? '—' }}</el-descriptions-item>
        <el-descriptions-item label="同步时间">{{ formatTime(data?.last_sync_at) }}</el-descriptions-item>
        <el-descriptions-item label="状态更新">{{ formatTime(data?.updated_at) }}</el-descriptions-item>
      </el-descriptions>
      <p class="page-hint mt-12">
        仅同步渠道配置中勾选了「同步余额」的启用渠道；勾选「余额通知」的渠道才会触发通知规则。
      </p>
    </el-card>
  </div>
</template>

<style scoped>
.form-hint {
  margin-left: 12px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.mt-12 {
  margin-top: 12px;
}
</style>
