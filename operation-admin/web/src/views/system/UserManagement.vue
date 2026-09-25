<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { SysUserVO, SysRole } from '@/api/rbac'
import {
  getUsersApi,
  createUserApi,
  updateUserApi,
  deleteUserApi,
  getRolesApi,
} from '@/api/rbac'

const tableData = ref<SysUserVO[]>([])
const total = ref(0)
const loading = ref(false)
const roles = ref<SysRole[]>([])
const query = reactive({ page: 1, page_size: 10, username: '', status: '' })

const dialogVisible = ref(false)
const dialogTitle = ref('新增用户')
const isEdit = ref(false)
const editId = ref(0)
const formRef = ref()

const form = reactive({
  username: '',
  password: '',
  real_name: '',
  phone: '',
  email: '',
  status: 'active',
  role_ids: [] as number[],
})

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function fetchData() {
  loading.value = true
  try {
    const data = await getUsersApi(query)
    tableData.value = data.records || []
    total.value = data.total || 0
  } finally {
    loading.value = false
  }
}

async function loadRoles() {
  roles.value = (await getRolesApi()) || []
}

onMounted(() => {
  fetchData()
  loadRoles()
})

function handleAdd() {
  dialogTitle.value = '新增用户'
  isEdit.value = false
  editId.value = 0
  Object.assign(form, {
    username: '',
    password: '',
    real_name: '',
    phone: '',
    email: '',
    status: 'active',
    role_ids: [],
  })
  dialogVisible.value = true
}

function handleEdit(row: SysUserVO) {
  dialogTitle.value = '编辑用户'
  isEdit.value = true
  editId.value = row.id
  Object.assign(form, {
    username: row.username,
    password: '',
    real_name: row.real_name || '',
    phone: row.phone || '',
    email: row.email || '',
    status: row.status || 'active',
    role_ids: row.role_ids || [],
  })
  dialogVisible.value = true
}

async function handleSave() {
  if (!isEdit.value) {
    const valid = await formRef.value?.validate().catch(() => false)
    if (!valid) return
  }
  const payload: Record<string, unknown> = {
    username: form.username,
    real_name: form.real_name,
    phone: form.phone,
    email: form.email,
    status: form.status,
    role_ids: form.role_ids,
  }
  if (form.password) payload.password = form.password
  if (isEdit.value) {
    await updateUserApi(editId.value, payload)
  } else {
    await createUserApi(payload)
  }
  ElMessage.success('保存成功')
  dialogVisible.value = false
  fetchData()
}

async function handleDelete(row: SysUserVO) {
  await ElMessageBox.confirm(`确定删除用户「${row.username}」？`, '提示', { type: 'warning' })
  await deleteUserApi(row.id)
  ElMessage.success('已删除')
  fetchData()
}
</script>

<template>
  <div class="page-container">
    <el-card shadow="never">
      <div class="toolbar mb-16">
        <el-input v-model="query.username" placeholder="用户名" clearable style="width: 180px" />
        <el-select v-model="query.status" placeholder="状态" clearable style="width: 120px">
          <el-option label="启用" value="active" />
          <el-option label="停用" value="inactive" />
        </el-select>
        <el-button type="primary" @click="() => { query.page = 1; fetchData() }">查询</el-button>
        <el-button v-permission="'system:user:create'" type="primary" @click="handleAdd">新增</el-button>
      </div>

      <el-table :data="tableData" v-loading="loading" stripe border size="small">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="username" label="用户名" width="140" />
        <el-table-column prop="real_name" label="姓名" width="120" />
        <el-table-column label="角色" min-width="160">
          <template #default="{ row }">{{ (row.role_names || []).join(', ') }}</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
              {{ row.status === 'active' ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'system:user:edit'" type="primary" link @click="handleEdit(row)">编辑</el-button>
            <el-button
              v-if="row.username !== 'admin'"
              v-permission="'system:user:delete'"
              type="danger"
              link
              @click="handleDelete(row)"
            >删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager mt-16">
        <el-pagination
          v-model:current-page="query.page"
          v-model:page-size="query.page_size"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="fetchData"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="520px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="isEdit ? {} : rules" label-width="90px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="密码" :prop="isEdit ? undefined : 'password'">
          <el-input v-model="form.password" type="password" show-password :placeholder="isEdit ? '留空则不修改' : ''" />
        </el-form-item>
        <el-form-item label="姓名"><el-input v-model="form.real_name" /></el-form-item>
        <el-form-item label="手机"><el-input v-model="form.phone" /></el-form-item>
        <el-form-item label="邮箱"><el-input v-model="form.email" /></el-form-item>
        <el-form-item label="角色">
          <el-select v-model="form.role_ids" multiple style="width: 100%">
            <el-option v-for="r in roles" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
        </el-form-item>
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
.pager {
  display: flex;
  justify-content: flex-end;
}
</style>
