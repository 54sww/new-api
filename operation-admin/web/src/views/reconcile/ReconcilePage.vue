<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import BalanceQueryDialog from '@/components/BalanceQueryDialog.vue'
import type { RechargeRow, ReconcileResult, ReconcileRow } from '@/types'
import { money } from '@/utils/format'
import {
  createRechargeApi,
  deleteRechargeApi,
  listRechargesApi,
  liveReconcileApi,
  refreshBalanceApi,
  syncBalancesApi,
  syncChannelsApi,
  syncStatusApi,
} from '@/api/finance'

const activeTab = ref('recon')
const loading = ref(false)
const data = ref<ReconcileResult | null>(null)
const recharges = ref<RechargeRow[]>([])
const syncStatus = ref<unknown>(null)
const bq = reactive({ open: false, id: 0, name: '' })
const rechargeForm = reactive({
  channel_id: 0,
  amount_usd: undefined as number | undefined,
  voucher: '',
  note: '',
})

let timer: number | undefined

const summary = computed(() => data.value?.summary)
const items = computed(() => data.value?.items || [])
const nameMap = computed(() =>
  Object.fromEntries(items.value.map((i) => [i.channel_id, i.name]))
)

async function refreshAll() {
  loading.value = true
  try {
    const live = await liveReconcileApi()
    data.value = live
    if (!rechargeForm.channel_id && live.items[0]) {
      rechargeForm.channel_id = live.items[0].channel_id
    }
    recharges.value = (await listRechargesApi()) || []
    syncStatus.value = await syncStatusApi()
  } finally {
    loading.value = false
  }
}

async function onSync() {
  try {
    await syncChannelsApi()
    await syncBalancesApi()
    await refreshAll()
    ElMessage.success('同步完成')
  } catch {
    // ignore
  }
}

async function onRefreshBalance(id: number) {
  try {
    await refreshBalanceApi(id)
    await refreshAll()
    ElMessage.success('余额已刷新')
  } catch {
    // ignore
  }
}

async function onAddRecharge() {
  if (!rechargeForm.channel_id || rechargeForm.amount_usd == null) {
    ElMessage.warning('请选择渠道并填写金额')
    return
  }
  try {
    await createRechargeApi({
      channel_id: rechargeForm.channel_id,
      amount_usd: Number(rechargeForm.amount_usd),
      voucher: rechargeForm.voucher,
      note: rechargeForm.note,
    })
    rechargeForm.amount_usd = undefined
    rechargeForm.voucher = ''
    rechargeForm.note = ''
    await refreshAll()
    ElMessage.success('充值已登记')
  } catch {
    // ignore
  }
}

async function onDeleteRecharge(id: number) {
  try {
    await ElMessageBox.confirm('确认删除该充值记录？', '删除', { type: 'warning' })
    await deleteRechargeApi(id)
    await refreshAll()
  } catch {
    // ignore
  }
}

function openBalanceConfig(row: ReconcileRow) {
  bq.open = true
  bq.id = row.channel_id
  bq.name = row.name
}

onMounted(() => {
  void refreshAll().catch(() => undefined)
  timer = window.setInterval(() => {
    void refreshAll().catch(() => undefined)
  }, 15000)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <div class="page-container" v-loading="loading">
    <div class="page-header mb-16">
      <div>
        <h2 class="page-title">实时对账</h2>
        <p class="page-hint">渠道余额与充值汇总；消耗统计见「消耗日报」</p>
      </div>
      <div class="page-actions">
        <el-button @click="refreshAll">刷新</el-button>
        <el-button v-permission="'finance:reconcile:sync'" type="primary" @click="onSync">立即同步</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="mb-16">
      <el-col :xs="12" :sm="8" :md="8" v-for="card in [
        { label: '渠道数', value: String(summary?.channel_count ?? 0) },
        { label: '余额合计', value: money(summary?.upstream_total) },
        { label: '充值合计', value: money(summary?.recharge_total) },
      ]" :key="card.label">
        <el-card shadow="never" class="stat-card">
          <div class="stat-label">{{ card.label }}</div>
          <div class="stat-value amount">{{ card.value }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="对账看板" name="recon">
          <el-table :data="items" stripe border height="60vh" size="small">
            <el-table-column prop="channel_id" label="ID" width="70" />
            <el-table-column label="渠道" min-width="200">
              <template #default="{ row }">
                {{ row.name }}
                <el-tag size="small" class="ml-8" :type="row.has_balance_query ? 'success' : 'info'">
                  {{ row.has_balance_query ? '自定义' : 'new-api' }}
                </el-tag>
                <el-tag v-if="row.sync_balance" size="small" class="ml-8" type="warning">同步</el-tag>
                <el-tag v-if="row.notify_enabled" size="small" class="ml-8" type="danger">通知</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="余额" min-width="140" align="right">
              <template #default="{ row }">
                <span class="amount">{{ money(row.upstream_balance) }}</span>
                <div
                  v-if="row.balance_currency && row.balance_currency !== 'USD'"
                  class="text-muted"
                  style="font-size: 12px"
                >
                  {{ row.balance_currency }} {{ Number(row.balance_raw || 0).toFixed(2) }}
                </div>
              </template>
            </el-table-column>
            <el-table-column label="充值金额" min-width="120" align="right">
              <template #default="{ row }">
                <span class="amount">{{ money(row.recharge_total) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="200" fixed="right">
              <template #default="{ row }">
                <el-button v-permission="'finance:balance:config'" link type="primary" @click="openBalanceConfig(row)">配置</el-button>
                <el-button v-permission="'finance:balance:refresh'" link type="primary" @click="onRefreshBalance(row.channel_id)">刷新余额</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="充值登记" name="recharge">
          <div class="recharge-form mb-16">
            <el-select v-model="rechargeForm.channel_id" placeholder="渠道" style="width: 220px">
              <el-option
                v-for="i in items"
                :key="i.channel_id"
                :label="`#${i.channel_id} ${i.name}`"
                :value="i.channel_id"
              />
            </el-select>
            <el-input-number
              v-model="rechargeForm.amount_usd"
              :controls="false"
              placeholder="金额 USD"
              style="width: 140px"
            />
            <el-input v-model="rechargeForm.voucher" placeholder="凭证号" style="width: 160px" />
            <el-input v-model="rechargeForm.note" placeholder="备注" style="flex: 1; min-width: 160px" />
            <el-button v-permission="'finance:recharge:create'" type="primary" @click="onAddRecharge">登记充值</el-button>
          </div>

          <el-table :data="recharges" stripe border height="50vh" size="small">
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column label="渠道" min-width="160">
              <template #default="{ row }">
                #{{ row.channel_id }} {{ nameMap[row.channel_id] || '' }}
              </template>
            </el-table-column>
            <el-table-column label="金额 USD" min-width="110" align="right">
              <template #default="{ row }">
                <span class="amount">{{ money(row.amount_usd) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="时间" min-width="160">
              <template #default="{ row }">
                {{ new Date(row.recharged_at * 1000).toLocaleString() }}
              </template>
            </el-table-column>
            <el-table-column prop="voucher" label="凭证" min-width="120" />
            <el-table-column prop="note" label="备注" min-width="140" />
            <el-table-column prop="created_by" label="操作人" width="100" />
            <el-table-column label="操作" width="80" fixed="right">
              <template #default="{ row }">
                <el-button v-permission="'finance:recharge:delete'" link type="danger" @click="onDeleteRecharge(row.id)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="同步状态" name="sync">
          <pre class="sync-box">{{ JSON.stringify(syncStatus, null, 2) }}</pre>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <BalanceQueryDialog
      v-model="bq.open"
      :channel-id="bq.id"
      :channel-name="bq.name"
      @saved="refreshAll"
    />
  </div>
</template>

<style scoped lang="scss">
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.page-hint {
  margin-top: 4px;
  font-size: 13px;
  color: $info-color;
}

.page-actions {
  display: flex;
  gap: 8px;
}

.stat-card {
  margin-bottom: 8px;

  :deep(.el-card__body) {
    padding: 12px 14px;
  }
}

.stat-label {
  font-size: 12px;
  color: $info-color;
}

.stat-value {
  margin-top: 6px;
  font-size: 18px;
}

.recharge-form {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.sync-box {
  white-space: pre-wrap;
  font-size: 13px;
  color: $info-color;
  background: #f5f7fa;
  padding: 12px;
  border-radius: $border-radius;
  max-height: 60vh;
  overflow: auto;
}

.ml-8 {
  margin-left: 8px;
}
</style>
