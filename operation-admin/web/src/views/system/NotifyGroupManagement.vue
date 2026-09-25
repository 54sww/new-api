<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { NotifyChannel, NotifyGroup } from '@/api/notify'
import {
  listNotifyChannelsApi,
  listNotifyGroupsApi,
  createNotifyGroupApi,
  updateNotifyGroupApi,
  deleteNotifyGroupApi,
} from '@/api/notify'

const tableData = ref<NotifyGroup[]>([])
const channels = ref<NotifyChannel[]>([])
const loading = ref(false)
const query = reactive({ name: '' })

const dialogVisible = ref(false)
const dialogTitle = ref('新增通知组')
const isEdit = ref(false)
const editId = ref(0)
const formRef = ref()

const form = reactive({
  name: '',
  code: '',
  status: 'active',
  remark: '',
  channel_ids: [] as number[],
})

const rules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
}

async function fetchData() {
  loading.value = true
  try {
    tableData.value = (await listNotifyGroupsApi({ name: query.name || undefined })) || []
  } finally {
    loading.value = false
  }
}

async function loadChannels() {
  channels.value = (await listNotifyChannelsApi({ status: 'active' })) || []
}

onMounted(() => {
  fetchData()
  loadChannels()
})

function handleAdd() {
  dialogTitle.value = '新增通知组'
  isEdit.value = false
  editId.value = 0
  Object.assign(form, { name: '', code: '', status: 'active', remark: '', channel_ids: [] })
  dialogVisible.value = true
}

function handleEdit(row: NotifyGroup) {
  dialogTitle.value = '编辑通知组'
  isEdit.value = true
  editId.value = row.id
  Object.assign(form, {
    name: row.name,
    code: row.code || '',
    status: row.status || 'active',
    remark: row.remark || '',
    channel_ids: row.channel_ids || [],
  })
  dialogVisible.value = true
}

async function handleSave() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  const payload = { ...form }
  if (isEdit.value) {
    await updateNotifyGroupApi(editId.value, payload)
  } else {
    await createNotifyGroupApi(payload)
  }
  ElMessage.success('保存成功')
  dialogVisible.value = false
  fetchData()
}

async function handleDelete(row: NotifyGroup) {
  await ElMessageBox.confirm(`确定删除通知组「${row.name}」？`, '提示', { type: 'warning' })
  await deleteNotifyGroupApi(row.id)
  ElMessage.success('已删除')
  fetchData()
}
</script>

<template>
  <div class="page-container">
    <el-card shadow="never">
      <div class="toolbar mb-16">
        <el-input v-model="query.name" placeholder="名称" clearable style="width: 180px" />
        <el-button type="primary" @click="fetchData">查询</el-button>
        <el-button v-permission="'system:notify-group:create'" type="primary" @click="handleAdd">新增</el-button>
      </div>

      <el-table :data="tableData" v-loading="loading" stripe border size="small">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="code" label="编码" width="140" />
        <el-table-column label="通知渠道" min-width="200">
          <template #default="{ row }">
            {{ (row.channel_names || []).join('、') || '—' }}
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="140" />
        <el-table-column prop="status" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
              {{ row.status === 'active' ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'system:notify-group:edit'" type="primary" link @click="handleEdit(row)">编辑</el-button>
            <el-button v-permission="'system:notify-group:delete'" type="danger" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="520px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="名称" prop="name"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="编码">
          <el-input v-model="form.code" placeholder="如 finance_alert，可选" />
        </el-form-item>
        <el-form-item label="通知渠道">
          <el-select v-model="form.channel_ids" multiple filterable style="width: 100%" placeholder="选择一个或多个渠道">
            <el-option
              v-for="ch in channels"
              :key="ch.id"
              :label="`${ch.name}（${ch.type === 'dingtalk' ? '钉钉' : '邮件'}）`"
              :value="ch.id"
            />
          </el-select>
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
</style>
