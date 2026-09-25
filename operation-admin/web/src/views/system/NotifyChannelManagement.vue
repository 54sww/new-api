<script setup lang="ts">
import { reactive, ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { NotifyChannel } from '@/api/notify'
import {
  listNotifyChannelsApi,
  createNotifyChannelApi,
  updateNotifyChannelApi,
  deleteNotifyChannelApi,
} from '@/api/notify'

const tableData = ref<NotifyChannel[]>([])
const loading = ref(false)
const query = reactive({ name: '', type: '', status: '' })

const dialogVisible = ref(false)
const dialogTitle = ref('新增通知渠道')
const isEdit = ref(false)
const editId = ref(0)
const formRef = ref()

const form = reactive({
  name: '',
  type: 'email' as string,
  status: 'active',
  remark: '',
  // email
  smtp_host: '',
  smtp_port: 465,
  username: '',
  password: '',
  from: '',
  tls: true,
  to: '',
  // dingtalk
  webhook: '',
  secret: '',
})

const rules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
}

const typeLabel = computed(() => (t: string) => (t === 'dingtalk' ? '钉钉' : t === 'email' ? '邮件' : t))

async function fetchData() {
  loading.value = true
  try {
    tableData.value = (await listNotifyChannelsApi({ ...query })) || []
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)

function resetForm() {
  Object.assign(form, {
    name: '',
    type: 'email',
    status: 'active',
    remark: '',
    smtp_host: '',
    smtp_port: 465,
    username: '',
    password: '',
    from: '',
    tls: true,
    to: '',
    webhook: '',
    secret: '',
  })
}

function handleAdd() {
  dialogTitle.value = '新增通知渠道'
  isEdit.value = false
  editId.value = 0
  resetForm()
  dialogVisible.value = true
}

function handleEdit(row: NotifyChannel) {
  dialogTitle.value = '编辑通知渠道'
  isEdit.value = true
  editId.value = row.id
  const cfg = row.config || {}
  Object.assign(form, {
    name: row.name,
    type: row.type,
    status: row.status || 'active',
    remark: row.remark || '',
    smtp_host: String(cfg.smtp_host || ''),
    smtp_port: Number(cfg.smtp_port || 465),
    username: String(cfg.username || ''),
    password: '',
    from: String(cfg.from || ''),
    tls: cfg.tls !== false,
    to: Array.isArray(cfg.to) ? (cfg.to as string[]).join(',') : String(cfg.to || ''),
    webhook: String(cfg.webhook || ''),
    secret: '',
  })
  dialogVisible.value = true
}

function buildConfig(): Record<string, unknown> {
  if (form.type === 'dingtalk') {
    return {
      webhook: form.webhook,
      secret: form.secret,
    }
  }
  const to = form.to
    .split(/[,;\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  return {
    smtp_host: form.smtp_host,
    smtp_port: form.smtp_port,
    username: form.username,
    password: form.password,
    from: form.from || form.username,
    tls: form.tls,
    to,
  }
}

async function handleSave() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  const payload = {
    name: form.name,
    type: form.type,
    status: form.status,
    remark: form.remark,
    config: buildConfig(),
  }
  if (isEdit.value) {
    await updateNotifyChannelApi(editId.value, payload)
  } else {
    await createNotifyChannelApi(payload)
  }
  ElMessage.success('保存成功')
  dialogVisible.value = false
  fetchData()
}

async function handleDelete(row: NotifyChannel) {
  await ElMessageBox.confirm(`确定删除通知渠道「${row.name}」？`, '提示', { type: 'warning' })
  await deleteNotifyChannelApi(row.id)
  ElMessage.success('已删除')
  fetchData()
}
</script>

<template>
  <div class="page-container">
    <el-card shadow="never">
      <div class="toolbar mb-16">
        <el-input v-model="query.name" placeholder="名称" clearable style="width: 160px" />
        <el-select v-model="query.type" placeholder="类型" clearable style="width: 120px">
          <el-option label="邮件" value="email" />
          <el-option label="钉钉" value="dingtalk" />
        </el-select>
        <el-select v-model="query.status" placeholder="状态" clearable style="width: 120px">
          <el-option label="启用" value="active" />
          <el-option label="停用" value="inactive" />
        </el-select>
        <el-button type="primary" @click="fetchData">查询</el-button>
        <el-button v-permission="'system:notify-channel:create'" type="primary" @click="handleAdd">新增</el-button>
      </div>

      <el-table :data="tableData" v-loading="loading" stripe border size="small">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">{{ typeLabel(row.type) }}</template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="160" />
        <el-table-column prop="status" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
              {{ row.status === 'active' ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'system:notify-channel:edit'" type="primary" link @click="handleEdit(row)">编辑</el-button>
            <el-button v-permission="'system:notify-channel:delete'" type="danger" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="560px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="名称" prop="name"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="类型" prop="type">
          <el-radio-group v-model="form.type" :disabled="isEdit">
            <el-radio value="email">邮件</el-radio>
            <el-radio value="dingtalk">钉钉</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio value="active">启用</el-radio>
            <el-radio value="inactive">停用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>

        <template v-if="form.type === 'email'">
          <el-form-item label="SMTP Host"><el-input v-model="form.smtp_host" placeholder="smtp.example.com" /></el-form-item>
          <el-form-item label="SMTP Port"><el-input-number v-model="form.smtp_port" :min="1" :max="65535" /></el-form-item>
          <el-form-item label="账号"><el-input v-model="form.username" /></el-form-item>
          <el-form-item label="密码">
            <el-input v-model="form.password" type="password" show-password :placeholder="isEdit ? '留空则不修改' : ''" />
          </el-form-item>
          <el-form-item label="发件人"><el-input v-model="form.from" placeholder="默认与账号相同" /></el-form-item>
          <el-form-item label="默认收件人">
            <el-input v-model="form.to" placeholder="多个用逗号分隔" />
          </el-form-item>
          <el-form-item label="TLS"><el-switch v-model="form.tls" /></el-form-item>
        </template>

        <template v-else>
          <el-form-item label="Webhook">
            <el-input v-model="form.webhook" type="textarea" :rows="2" placeholder="钉钉机器人 Webhook" />
          </el-form-item>
          <el-form-item label="加签 Secret">
            <el-input v-model="form.secret" type="password" show-password :placeholder="isEdit ? '留空则不修改' : '可选'" />
          </el-form-item>
        </template>
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
</style>
