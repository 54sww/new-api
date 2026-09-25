<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { BalanceQueryConfig } from '@/types'
import {
  balanceQueryPresetsApi,
  getBalanceQueryApi,
  putBalanceQueryApi,
  refreshBalanceApi,
  testBalanceQueryApi,
} from '@/api/finance'

const props = defineProps<{
  modelValue: boolean
  channelId: number
  channelName: string
}>()

const emit = defineEmits<{
  'update:modelValue': [boolean]
  saved: []
}>()

const loading = ref(false)
const channelBaseUrl = ref('')
const result = ref('')
  const form = reactive({
  sync_balance: false,
  notify_enabled: false,
  alias: '',
  enabled: false,
  base_url: '',
  url: '{{base_url}}/api/user/self',
  method: 'GET',
  amount_path: 'data.quota',
  currency_path: '',
  currency: 'USD',
  fx_to_usd: 0.000002,
  headersText: JSON.stringify({ Authorization: 'Bearer {token}' }, null, 2),
  body: '',
  api_key: '',
})

const currencyOptions = ['USD', 'CNY', 'EUR', 'GBP', 'HKD', 'JPY', 'SGD', 'AUD', 'CAD', 'KRW', 'TWD']

watch(
  () => [props.modelValue, props.channelId] as const,
  async ([open]) => {
    if (!open) return
    loading.value = true
    result.value = ''
    form.api_key = ''
    try {
      const data = await getBalanceQueryApi(props.channelId)
      const cfg = data.config || {}
      const chBase = (data.channel_base_url || '').replace(/\/+$/, '')
      channelBaseUrl.value = chBase
      form.sync_balance = !!data.sync_balance
      form.notify_enabled = !!data.notify_enabled
      form.alias = data.alias || ''
      form.enabled = !!cfg.enabled
      form.base_url = (cfg.base_url || chBase || '').replace(/\/+$/, '')
      form.url = cfg.url || '{{base_url}}/api/user/self'
      form.method = (cfg.method || 'GET').toUpperCase()
      form.amount_path = cfg.amount_path || 'data.quota'
      form.currency_path = cfg.currency_path || ''
      form.currency = cfg.currency || 'USD'
      if (form.currency && !currencyOptions.includes(form.currency)) {
        currencyOptions.push(form.currency)
      }
      form.fx_to_usd = cfg.fx_to_usd ?? 0.000002
      form.headersText = JSON.stringify(
        cfg.headers || { Authorization: 'Bearer {token}' },
        null,
        2
      )
      form.body = cfg.body || ''
      result.value = chBase
        ? `已从渠道 base_url 带入：${chBase}`
        : '该渠道暂无 base_url，请先在 new-api 配置并同步'
    } catch {
      // 拦截器已提示
    } finally {
      loading.value = false
    }
  },
  { immediate: true }
)

function readConfig(): BalanceQueryConfig {
  let headers: Record<string, string> = {}
  try {
    headers = JSON.parse(form.headersText || '{}') as Record<string, string>
  } catch {
    throw new Error('Headers 不是合法 JSON')
  }
  return {
    enabled: form.enabled,
    base_url: form.base_url.trim(),
    url: form.url.trim(),
    method: form.method,
    amount_path: form.amount_path.trim(),
    currency_path: form.currency_path.trim(),
    currency: form.currency.trim(),
    fx_to_usd: Number(form.fx_to_usd || 0),
    headers,
    body: form.body,
    api_key: form.api_key,
    timeout_sec: 30,
  }
}

async function applyPreset() {
  try {
    const presets = await balanceQueryPresetsApi()
    const p = presets.find((x) => x.id === 'newapi') || presets[0]
    if (!p) return
    const base = (form.base_url || channelBaseUrl.value || '').replace(/\/+$/, '')
    if (!base) {
      ElMessage.error('该渠道没有 base_url，请先在 new-api 配置渠道地址并点「立即同步」')
      return
    }
    const cfg = { ...(p.config || {}), base_url: base }
    form.enabled = true
    form.base_url = base
    form.url = cfg.url || '{{base_url}}/api/user/self'
    form.method = (cfg.method || 'GET').toUpperCase()
    form.amount_path = cfg.amount_path || 'data.quota'
    form.currency_path = cfg.currency_path || ''
    form.currency = cfg.currency || 'USD'
    form.fx_to_usd = cfg.fx_to_usd ?? 0.000002
    form.headersText = JSON.stringify(
      cfg.headers || { Authorization: 'Bearer {token}' },
      null,
      2
    )
    form.body = cfg.body || ''
    ElMessage.success(`已套用 New API 模板（Base URL: ${base}）`)
  } catch {
    // ignore
  }
}

async function onTest() {
  try {
    const cfg = readConfig()
    const res = await testBalanceQueryApi(props.channelId, cfg)
    result.value = `成功: raw=${res.amount_raw} ${res.currency} → USD ${res.amount_usd}\n${res.raw_body || ''}`
  } catch (err) {
    result.value = '失败: ' + (err instanceof Error ? err.message : 'error')
  }
}

async function onSave(refresh: boolean) {
  try {
    const cfg = readConfig()
    await putBalanceQueryApi(props.channelId, {
      sync_balance: form.sync_balance,
      notify_enabled: form.notify_enabled,
      alias: form.alias.trim(),
      config: cfg,
    })
    if (refresh) {
      const res = await refreshBalanceApi(props.channelId)
      result.value = `已刷新: USD ${res.balance} (raw ${res.balance_raw} ${res.balance_currency || ''})`
    }
    ElMessage.success('已保存')
    emit('saved')
  } catch (err) {
    if (err instanceof Error && err.message.includes('Headers')) {
      ElMessage.error(err.message)
    }
  }
}

function close() {
  emit('update:modelValue', false)
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    :title="`渠道配置 #${channelId} ${channelName}`"
    width="720px"
    destroy-on-close
    @close="close"
  >
    <div v-loading="loading">
      <el-alert
        class="mb-16"
        type="info"
        :closable="false"
        title="测试渠道请关闭「同步余额」与「余额通知」，避免定时刷新与告警噪音。"
      />

      <el-form label-position="top" class="mb-16">
        <el-form-item label="渠道别名">
          <el-input
            v-model="form.alias"
            maxlength="64"
            show-word-limit
            placeholder="通知时显示此名称；留空则用原渠道名"
          />
          <div class="field-hint">余额告警消息里会用别名代替 new-api 渠道名</div>
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="同步余额">
              <el-switch
                v-model="form.sync_balance"
                active-text="开启"
                inactive-text="关闭"
              />
              <div class="field-hint">关闭后不参与定时/批量余额刷新（手动刷新仍可用）</div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="余额通知">
              <el-switch
                v-model="form.notify_enabled"
                active-text="开启"
                inactive-text="关闭"
              />
              <div class="field-hint">开启后纳入后续余额/差额通知（测试渠道建议关闭）</div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <el-divider content-position="left">余额查询</el-divider>

      <div class="toolbar mb-16">
        <el-button @click="applyPreset">套用 New API 模板</el-button>
        <el-checkbox v-model="form.enabled">启用自定义查询</el-checkbox>
      </div>

      <el-form label-width="140px" label-position="top">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="Base URL（来自渠道 base_url）">
              <el-input v-model="form.base_url" readonly />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="URL">
              <el-input v-model="form.url" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Method">
              <el-select v-model="form.method" style="width: 100%">
                <el-option label="GET" value="GET" />
                <el-option label="POST" value="POST" />
                <el-option label="PUT" value="PUT" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="金额 JSON Path">
              <el-input v-model="form.amount_path" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="币种 JSON Path（可选）">
              <el-input v-model="form.currency_path" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="默认币种">
              <el-select v-model="form.currency" filterable style="width: 100%">
                <el-option
                  v-for="c in currencyOptions"
                  :key="c"
                  :label="c"
                  :value="c"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="换算到 USD（1 USD = 500000 Quota → 0.000002）">
              <el-input-number
                v-model="form.fx_to_usd"
                :step="0.0000001"
                :controls="false"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="Headers JSON（支持 {token}）">
          <el-input v-model="form.headersText" type="textarea" :rows="4" />
        </el-form-item>
        <el-form-item label="Body（可选）">
          <el-input v-model="form.body" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="Token（替换 {token}，留空则保留原值）">
          <el-input v-model="form.api_key" type="password" show-password />
        </el-form-item>
      </el-form>

      <pre v-if="result" class="result-box">{{ result }}</pre>
    </div>

    <template #footer>
      <el-button @click="onTest">测试拉取</el-button>
      <el-button type="primary" @click="onSave(false)">保存</el-button>
      <el-button type="success" @click="onSave(true)">保存并刷新</el-button>
      <el-button @click="close">关闭</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.toolbar {
  display: flex;
  align-items: center;
  gap: 16px;
}

.field-hint {
  margin-top: 4px;
  font-size: 12px;
  color: $info-color;
  line-height: 1.4;
}

.result-box {
  margin-top: 12px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: $border-radius;
  white-space: pre-wrap;
  font-size: 12px;
  color: $info-color;
  max-height: 180px;
  overflow: auto;
}
</style>
