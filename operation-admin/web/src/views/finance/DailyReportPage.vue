<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  dailyReportApi,
  dailySnapshotsApi,
  liveReconcileApi,
  type DailyReportResult,
  type DailyReportRow,
  type SnapshotPoint,
} from '@/api/finance'
import { money } from '@/utils/format'

function todayStr() {
  const d = new Date()
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function daysAgoStr(n: number) {
  const d = new Date()
  d.setDate(d.getDate() - n)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function fmtTs(ts?: number) {
  if (!ts) return '—'
  return new Date(ts * 1000).toLocaleString()
}

const loading = ref(false)
const data = ref<DailyReportResult | null>(null)
const channels = ref<Array<{ id: number; name: string }>>([])
const activeTab = ref('detail')
const statusFilter = ref<'all' | 'ok' | 'partial'>('all')

const query = reactive({
  range: [daysAgoStr(6), todayStr()] as [string, string],
  channel_id: undefined as number | undefined,
  only_sync: true,
})

const items = computed(() => {
  const list = data.value?.items || []
  if (statusFilter.value === 'ok') {
    return list.filter((i) => i.status === 'ok' || i.status === 'in_progress')
  }
  if (statusFilter.value === 'partial') {
    return list.filter((i) => i.status === 'partial')
  }
  return list
})
const byDay = computed(() => data.value?.by_day || [])
const summary = computed(() => data.value?.summary)

const drawerOpen = ref(false)
const drawerTitle = ref('')
const drawerLoading = ref(false)
const snapshots = ref<SnapshotPoint[]>([])

const statusMap: Record<string, { label: string; type: 'success' | 'warning' | 'info' | 'danger' }> = {
  ok: { label: '完整', type: 'success' },
  in_progress: { label: '进行中', type: 'info' },
  partial: { label: '不完整', type: 'warning' },
  no_data: { label: '无数据', type: 'danger' },
}

async function loadChannels() {
  try {
    const live = await liveReconcileApi()
    channels.value = (live.items || []).map((i) => ({ id: i.channel_id, name: i.name }))
  } catch {
    channels.value = []
  }
}

async function fetchData() {
  loading.value = true
  try {
    const [from, to] = query.range || [todayStr(), todayStr()]
    data.value = await dailyReportApi({
      from,
      to,
      channel_id: query.channel_id,
      only_sync: query.only_sync,
    })
  } finally {
    loading.value = false
  }
}

async function openSnapshots(row: DailyReportRow) {
  drawerTitle.value = `${row.channel_name}（#${row.channel_id}）· ${row.day_key}`
  drawerOpen.value = true
  drawerLoading.value = true
  snapshots.value = []
  try {
    snapshots.value = (await dailySnapshotsApi({ channel_id: row.channel_id, day: row.day_key })) || []
  } finally {
    drawerLoading.value = false
  }
}

function setPreset(days: number) {
  query.range = [daysAgoStr(days - 1), todayStr()]
  void fetchData()
}

onMounted(() => {
  void loadChannels()
  void fetchData()
})
</script>

<template>
  <div class="page-container" v-loading="loading">
    <div class="page-header mb-16">
      <div>
        <h2 class="page-title">消耗日报</h2>
        <p class="page-hint">
          日初 = 当天 00:00 前最近快照；消耗 = 日初 + 当日充值 − 日末。需开启自动同步并勾选渠道「同步余额」。
        </p>
      </div>
      <div class="page-actions filters">
        <el-date-picker
          v-model="query.range"
          type="daterange"
          value-format="YYYY-MM-DD"
          start-placeholder="开始"
          end-placeholder="结束"
          :clearable="false"
          style="width: 260px"
        />
        <el-select
          v-model="query.channel_id"
          clearable
          filterable
          placeholder="全部渠道"
          style="width: 180px"
        >
          <el-option v-for="c in channels" :key="c.id" :label="`${c.name} (#${c.id})`" :value="c.id" />
        </el-select>
        <el-checkbox v-model="query.only_sync">仅同步余额渠道</el-checkbox>
        <el-button-group>
          <el-button @click="setPreset(1)">今天</el-button>
          <el-button @click="setPreset(7)">近7天</el-button>
          <el-button @click="setPreset(30)">近30天</el-button>
        </el-button-group>
        <el-button type="primary" @click="fetchData">查询</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="mb-16">
      <el-col
        :xs="12"
        :sm="8"
        :md="4"
        v-for="card in [
          { label: '消耗合计', value: money(summary?.consume_total) },
          { label: '充值合计', value: money(summary?.recharge_total) },
          { label: '渠道数', value: String(summary?.channel_count ?? 0) },
          { label: '完整行', value: String(summary?.ok_count ?? 0) },
          { label: '不完整', value: String(summary?.partial_count ?? 0) },
          { label: '进行中', value: String(summary?.in_progress_count ?? 0) },
        ]"
        :key="card.label"
      >
        <el-card shadow="never" class="stat-card">
          <div class="stat-label">{{ card.label }}</div>
          <div class="stat-value amount">{{ card.value }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="按日汇总" name="byday">
          <el-table :data="byDay" stripe border size="small" empty-text="暂无数据">
            <el-table-column prop="day_key" label="日期" width="120" fixed />
            <el-table-column prop="channel_count" label="渠道" width="80" align="right" />
            <el-table-column label="消耗" min-width="120" align="right">
              <template #default="{ row }">
                <span class="amount">{{ money(row.consume_total) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="充值" min-width="120" align="right">
              <template #default="{ row }">{{ money(row.recharge_total) }}</template>
            </el-table-column>
            <el-table-column label="日初合计" min-width="120" align="right">
              <template #default="{ row }">{{ money(row.start_total) }}</template>
            </el-table-column>
            <el-table-column label="日末合计" min-width="120" align="right">
              <template #default="{ row }">{{ money(row.end_total) }}</template>
            </el-table-column>
            <el-table-column label="完整 / 不完整 / 进行中" min-width="160" align="center">
              <template #default="{ row }">
                {{ row.ok_count }} / {{ row.partial_count }} / {{ row.in_progress_count }}
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="渠道明细" name="detail">
          <div class="mb-8">
            <el-radio-group v-model="statusFilter" size="small">
              <el-radio-button value="all">全部</el-radio-button>
              <el-radio-button value="ok">完整+进行中</el-radio-button>
              <el-radio-button value="partial">仅不完整</el-radio-button>
            </el-radio-group>
          </div>
          <el-table :data="items" stripe border size="small" height="52vh" empty-text="暂无快照，请先开启自动同步">
            <el-table-column prop="day_key" label="日期" width="110" fixed />
            <el-table-column prop="channel_id" label="ID" width="70" />
            <el-table-column prop="channel_name" label="渠道" min-width="140" show-overflow-tooltip />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tooltip :content="row.status_note" placement="top">
                  <el-tag :type="statusMap[row.status]?.type || 'info'" size="small">
                    {{ statusMap[row.status]?.label || row.status }}
                  </el-tag>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column label="日初余额" width="120" align="right">
              <template #default="{ row }">
                <div>{{ money(row.start_balance) }}</div>
                <div class="sub-ts">{{ fmtTs(row.start_at) }}</div>
              </template>
            </el-table-column>
            <el-table-column label="当日充值" width="110" align="right">
              <template #default="{ row }">{{ money(row.recharge_usd) }}</template>
            </el-table-column>
            <el-table-column label="日末余额" width="120" align="right">
              <template #default="{ row }">
                <div>{{ money(row.end_balance) }}</div>
                <div class="sub-ts">{{ fmtTs(row.end_at) }}</div>
              </template>
            </el-table-column>
            <el-table-column label="当日消耗" width="120" align="right" sortable :sort-method="(a: DailyReportRow, b: DailyReportRow) => a.consume_usd - b.consume_usd">
              <template #default="{ row }">
                <span class="amount" :class="{ 'text-danger': row.consume_usd < -0.01 }">
                  {{ money(row.consume_usd) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="snapshot_cnt" label="日内快照" width="90" align="right" />
            <el-table-column label="操作" width="90" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openSnapshots(row)">快照</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-drawer v-model="drawerOpen" :title="drawerTitle" size="480px">
      <div v-loading="drawerLoading">
        <p class="page-hint mb-8">含日初结转快照（source 含 /open）与当日所有余额采样。</p>
        <el-table :data="snapshots" size="small" border stripe empty-text="无快照">
          <el-table-column label="时间" min-width="160">
            <template #default="{ row }">{{ fmtTs(row.synced_at) }}</template>
          </el-table-column>
          <el-table-column label="余额 USD" width="110" align="right">
            <template #default="{ row }">{{ money(row.balance) }}</template>
          </el-table-column>
          <el-table-column prop="source" label="来源" width="90" />
          <el-table-column prop="currency" label="币种" width="70" />
        </el-table>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
.filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  justify-content: flex-end;
}
.sub-ts {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  font-weight: normal;
  margin-top: 2px;
}
.stat-card {
  margin-bottom: 0;
}
.stat-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.stat-value {
  margin-top: 6px;
  font-size: 18px;
}
</style>
