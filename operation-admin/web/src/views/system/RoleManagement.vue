<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { SysMenu, SysRole } from '@/api/rbac'
import {
  getRolesApi,
  createRoleApi,
  updateRoleApi,
  deleteRoleApi,
  getRoleDetailApi,
  assignRoleMenusApi,
  getMenuTreeApi,
} from '@/api/rbac'

const tableData = ref<SysRole[]>([])
const loading = ref(false)
const menuTree = ref<SysMenu[]>([])

const dialogVisible = ref(false)
const menuDialogVisible = ref(false)
const dialogTitle = ref('新增角色')
const isEdit = ref(false)
const editId = ref(0)
const currentRoleId = ref(0)
const menuTreeRef = ref()
const formRef = ref()

const form = reactive({ code: '', name: '', status: 'active', remark: '' })
const rules = {
  code: [{ required: true, message: '请输入角色编码', trigger: 'blur' }],
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
}

async function fetchData() {
  loading.value = true
  try {
    tableData.value = (await getRolesApi()) || []
  } finally {
    loading.value = false
  }
}

async function loadMenuTree() {
  menuTree.value = (await getMenuTreeApi()) || []
}

onMounted(() => {
  fetchData()
  loadMenuTree()
})

function handleAdd() {
  dialogTitle.value = '新增角色'
  isEdit.value = false
  editId.value = 0
  Object.assign(form, { code: '', name: '', status: 'active', remark: '' })
  dialogVisible.value = true
}

function handleEdit(row: SysRole) {
  dialogTitle.value = '编辑角色'
  isEdit.value = true
  editId.value = row.id
  Object.assign(form, {
    code: row.code,
    name: row.name,
    status: row.status || 'active',
    remark: row.remark || '',
  })
  dialogVisible.value = true
}

async function handleSave() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  if (isEdit.value) {
    await updateRoleApi(editId.value, form)
  } else {
    await createRoleApi(form)
  }
  ElMessage.success('保存成功')
  dialogVisible.value = false
  fetchData()
}

async function handleDeleteRow(row: SysRole) {
  await ElMessageBox.confirm(`确定删除角色「${row.name}」？`, '提示', { type: 'warning' })
  await deleteRoleApi(row.id)
  ElMessage.success('已删除')
  fetchData()
}

async function openMenuAssign(row: SysRole) {
  currentRoleId.value = row.id
  const detail = await getRoleDetailApi(row.id)
  menuDialogVisible.value = true
  setTimeout(() => {
    menuTreeRef.value?.setCheckedKeys(detail.menu_ids || [])
  }, 80)
}

async function saveMenuAssign() {
  const checked = (menuTreeRef.value?.getCheckedKeys(false) as number[]) || []
  const half = (menuTreeRef.value?.getHalfCheckedKeys() as number[]) || []
  const menuIds = [...new Set([...checked, ...half])]
  await assignRoleMenusApi(currentRoleId.value, menuIds)
  ElMessage.success('菜单权限已保存')
  menuDialogVisible.value = false
}
</script>

<template>
  <div class="page-container">
    <el-card shadow="never">
      <div class="toolbar mb-16">
        <el-button v-permission="'system:role:create'" type="primary" @click="handleAdd">新增</el-button>
      </div>

      <el-table :data="tableData" v-loading="loading" stripe border size="small">
        <el-table-column prop="code" label="角色编码" width="160" />
        <el-table-column prop="name" label="角色名称" width="160" />
        <el-table-column prop="remark" label="备注" min-width="200" />
        <el-table-column prop="status" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
              {{ row.status === 'active' ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'system:role:assign'" type="primary" link @click="openMenuAssign(row)">分配菜单</el-button>
            <el-button v-permission="'system:role:edit'" type="primary" link @click="handleEdit(row)">编辑</el-button>
            <el-button
              v-if="row.code !== 'admin'"
              v-permission="'system:role:delete'"
              type="danger"
              link
              @click="handleDeleteRow(row)"
            >删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="480px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="角色编码" prop="code">
          <el-input v-model="form.code" :disabled="isEdit && form.code === 'admin'" />
        </el-form-item>
        <el-form-item label="角色名称" prop="name"><el-input v-model="form.name" /></el-form-item>
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

    <el-dialog v-model="menuDialogVisible" title="分配菜单 / 按钮权限" width="520px">
      <el-tree
        ref="menuTreeRef"
        :data="menuTree"
        node-key="id"
        show-checkbox
        default-expand-all
        :props="{ label: 'name', children: 'children' }"
      />
      <template #footer>
        <el-button @click="menuDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveMenuAssign">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.toolbar {
  display: flex;
  gap: 8px;
}
</style>
