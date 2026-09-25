<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { NotifyGroup, NotifyRule } from '@/api/notify'
import {
  listNotifyGroupsApi,
  listNotifyRulesApi,
  createNotifyRuleApi,
  updateNotifyRuleApi,
  deleteNotifyRuleApi,
} from '@/api/notify'
import { liveReconcileApi } from '@/api/finance'

type ChannelOpt = { id: number; name: string; notify_enabled?: boolean }

const tableData = ref<NotifyRule[]>([])
const groups = ref<NotifyGroup[]>([])
const channels = ref<ChannelOpt[]>([])
const loading = ref(false)
const query = reactive({ name: '', status: '' })

const dialogVisible = ref(false)
const dialogTitle = ref('新增通知规则')
const isEdit = ref(false)
const editId = ref(0)
const formRef = ref()

const form = reactive({
  name: '',
  status: 'active',
  remark: '',
  event_type: 'balance_below',
  threshold_usd: 100,
  notify_group_id: undefined as number | undefined,
  scope: 'all_enabled',
  channel_ids: [] as number[],
  cooldown_minutes: 60,
})

const rules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  notify_group_id: [{ required: true, message: '请选择通知组', trigger: 'change' }],
  threshold_usd: [{ required: true, message: '请输入余额阈值', trigger: 'blur' }],
}

async function fetchData() {
  loading.value = true
  try {
    tableData.value =
      (await listNotifyRulesApi({
        name: query.name || undefined,
        status: query.status || undefined,
      })) || []
  } finally {
    loading.value = false
  }
}

async function loadOptions() {
  groups.value = (await listNotifyGroupsApi()) || []
  try {
    const live = await liveReconcileApi()
    channels.value = (live.items || []).map((i) => ({
      id: i.channel_id,
      name: i.name,
      notify_enabled: i.notify_enabled,
    }))
  } catch {
    channels.value = []
  }
}

onMounted(() => {
  fetchData()
  loadOptions()
})

function handleAdd() {
  dialogTitle.value = '新增通知规则'
  isEdit.value = false
  editId.value = 0
  Object.assign(form, {
    name: '',
    status: 'active',
    remark: '',
    event_type: 'balance_below',
    threshold_usd: 100,
    notify_group_id: groups.value[0]?.id,
    scope: 'all_enabled',
    channel_ids: [],
    cooldown_minutes: 60,
  })
  dialogVisible.value = true
}

function handleEdit(row: NotifyRule) {
  dialogTitle.value = '编辑通知规则'
  isEdit.value = true
  editId.value = row.id
  Object.assign(form, {
    name: row.name,
    status: row.status || 'active',
    remark: row.remark || '',
    event_type: row.event_type || 'balance_below',
    threshold_usd: row.threshold_usd ?? 100,
    notify_group_id: row.notify_group_id,
    scope: row.scope || 'all_enabled',
    channel_ids: row.channel_ids || [],
    cooldown_minutes: row.cooldown_minutes || 60,
  })
  dialogVisible.value = true
}

async function handleSave() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  const payload = {
    name: form.name,
    status: form.status,
    remark: form.remark,
    event_type: form.event_type,
    threshold_usd: form.threshold_usd,
    notify_group_id: form.notify_group_id,
    scope: form.scope,
    channel_ids: form.scope === 'selected' ? form.channel_ids : [],
    cooldown_minutes: form.cooldown_minutes,
  }
  if (isEdit.value) {
    await updateNotifyRuleApi(editId.value, payload)
  } else {
    await createNotifyRuleApi(payload)
  }
  ElMessage.success('保存成功')
  dialogVisible.value = false
  fetchData()
}

async function handleDelete(row: NotifyRule) {
  await ElMessageBox.confirm(`确定删除规则「${row.name}」？`, '提示', { type: 'warning' })
  await deleteNotifyRuleApi(row.id)
  ElMessage.success('已删除')
  fetchData()
}

function scopeLabel(scope?: string) {
  return scope === 'selected' ? '指定渠道' : '全部开启通知的渠道'
}
</script>

<template>
  <div class="page-container">
    <el-card shadow="never">
      <el-alert
        class="mb-16"
        type="info"
        :closable="false"
        title="当渠道开启「余额通知」且余额刷新后低于阈值时，将通过通知组发送告警（受冷却时间限制）。"
      />

      <div class="toolbar mb-16">
        <el-input v-model="query.name" placeholder="名称" clearable style="width: 160px" />
        <el-select v-model="query.status" placeholder="状态" clearable style="width: 120px">
          <el-option label="启用" value="active" />
          <el-option label="停用" value="inactive" />
        </el-select>
        <el-button type="primary" @click="fetchData">查询</el-button>
        <el-button v-permission="'system:notify-rule:create'" type="primary" @click="handleAdd">新增</el-button>
      </div>

      <el-table :data="tableData" v-loading="loading" stripe border size="small">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="规则名称" min-width="140" />
        <el-table-column label="条件" min-width="160">
          <template #default="{ row }">
            余额 &lt; {{ row.threshold_usd }} USD
          </template>
        </el-table-column>
        <el-table-column prop="notify_group_name" label="通知组" min-width="120" />
        <el-table-column label="范围" min-width="140">
          <template #default="{ row }">{{ scopeLabel(row.scope) }}</template>
        </el-table-column>
        <el-table-column label="冷却(分)" width="90" prop="cooldown_minutes" />
        <el-table-column prop="status" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
              {{ row.status === 'active' ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'system:notify-rule:edit'" type="primary" link @click="handleEdit(row)">编辑</el-button>
            <el-button v-permission="'system:notify-rule:delete'" type="danger" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="560px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-form-item label="规则名称" prop="name"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="余额阈值(USD)" prop="threshold_usd">
          <el-input-number v-model="form.threshold_usd" :min="0" :step="1" :controls="false" style="width: 100%" />
          <div class="hint">当渠道剩余余额低于该值时触发通知</div>
        </el-form-item>
        <el-form-item label="通知组" prop="notify_group_id">
          <el-select v-model="form.notify_group_id" style="width: 100%" placeholder="选择通知组">
            <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="适用渠道">
          <el-radio-group v-model="form.scope">
            <el-radio value="all_enabled">全部开启「余额通知」的渠道</el-radio>
            <el-radio value="selected">指定渠道</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="form.scope === 'selected'" label="渠道列表">
          <el-select v-model="form.channel_ids" multiple filterable style="width: 100%">
            <el-option
              v-for="ch in channels"
              :key="ch.id"
              :label="`#${ch.id} ${ch.name}${ch.notify_enabled ? '' : '（未开通知）'}`"
              :value="ch.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="冷却时间(分)">
          <el-input-number v-model="form.cooldown_minutes" :min="1" :max="10080" />
          <div class="hint">同一渠道同一规则在冷却期内不重复发送</div>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio value="active">启用</el-radio>
            <el-radio value="inactive">停用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.hint {
  margin-top: 4px;
  font-size: 12px;
  color: $info-color;
}
</style>
