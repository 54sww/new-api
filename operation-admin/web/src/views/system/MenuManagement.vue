<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { SysMenu } from '@/api/rbac'
import { getMenuTreeApi, createMenuApi, updateMenuApi, deleteMenuApi } from '@/api/rbac'

const menuTree = ref<SysMenu[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('新增菜单')
const isEdit = ref(false)
const editId = ref(0)
const formRef = ref()

const subsystemOptions = [
  { label: '公共', value: 'common' },
  { label: '财务管理', value: 'finance' },
  { label: '运维管理', value: 'ops' },
  { label: '系统管理', value: 'system' },
]

const form = reactive<Partial<SysMenu>>({
  parent_id: null,
  name: '',
  path: '',
  permission: '',
  icon: '',
  parent_icon: '',
  subsystem: 'system',
  menu_type: 'C',
  menu_group: '',
  sort: 1,
  status: 'active',
  hidden: false,
})

const rules = { name: [{ required: true, message: '请输入菜单名称', trigger: 'blur' }] }

async function fetchData() {
  loading.value = true
  try {
    menuTree.value = (await getMenuTreeApi()) || []
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)

function handleAdd(parent?: SysMenu) {
  dialogTitle.value = '新增菜单'
  isEdit.value = false
  editId.value = 0
  Object.assign(form, {
    parent_id: parent?.id || null,
    name: '',
    path: '',
    permission: '',
    icon: '',
    parent_icon: '',
    subsystem: parent?.subsystem || 'system',
    menu_type: parent ? 'C' : 'M',
    menu_group: parent?.menu_group || parent?.name || '',
    sort: 1,
    status: 'active',
    hidden: false,
  })
  dialogVisible.value = true
}

function handleEdit(row: SysMenu) {
  dialogTitle.value = '编辑菜单'
  isEdit.value = true
  editId.value = row.id
  Object.assign(form, { ...row })
  dialogVisible.value = true
}

async function handleSave() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  if (isEdit.value) {
    await updateMenuApi(editId.value, form)
  } else {
    await createMenuApi(form)
  }
  ElMessage.success('保存成功')
  dialogVisible.value = false
  fetchData()
}

async function handleDelete(row: SysMenu) {
  await ElMessageBox.confirm(`确定删除菜单「${row.name}」？`, '提示', { type: 'warning' })
  await deleteMenuApi(row.id)
  ElMessage.success('已删除')
  fetchData()
}
</script>

<template>
  <div class="page-container">
    <el-card shadow="never">
      <div class="toolbar mb-16">
        <el-button v-permission="'system:menu:create'" type="primary" @click="handleAdd()">新增</el-button>
      </div>

      <el-table
        :data="menuTree"
        v-loading="loading"
        row-key="id"
        border
        size="small"
        default-expand-all
        :tree-props="{ children: 'children' }"
      >
        <el-table-column prop="name" label="菜单名称" min-width="160" />
        <el-table-column prop="menu_type" label="类型" width="80">
          <template #default="{ row }">
            {{ row.menu_type === 'M' ? '目录' : row.menu_type === 'F' ? '按钮' : '菜单' }}
          </template>
        </el-table-column>
        <el-table-column prop="path" label="路径" min-width="160" />
        <el-table-column prop="permission" label="权限标识" min-width="180" />
        <el-table-column prop="subsystem" label="子系统" width="100" />
        <el-table-column prop="sort" label="排序" width="70" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.menu_type !== 'F'"
              v-permission="'system:menu:create'"
              type="primary"
              link
              @click="handleAdd(row)"
            >添加子项</el-button>
            <el-button v-permission="'system:menu:edit'" type="primary" link @click="handleEdit(row)">编辑</el-button>
            <el-button v-permission="'system:menu:delete'" type="danger" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="560px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="菜单名称" prop="name"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="form.menu_type">
            <el-radio value="M">目录</el-radio>
            <el-radio value="C">菜单</el-radio>
            <el-radio value="F">按钮</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="子系统">
          <el-select v-model="form.subsystem" style="width: 100%">
            <el-option v-for="o in subsystemOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.menu_type === 'C'" label="路由路径">
          <el-input v-model="form.path" placeholder="/system/users" />
        </el-form-item>
        <el-form-item label="权限标识">
          <el-input v-model="form.permission" placeholder="system:user:list" />
        </el-form-item>
        <el-form-item label="图标"><el-input v-model="form.icon" placeholder="Element Plus 图标名" /></el-form-item>
        <el-form-item label="分组图标"><el-input v-model="form.parent_icon" /></el-form-item>
        <el-form-item label="菜单分组"><el-input v-model="form.menu_group" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" /></el-form-item>
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
  gap: 8px;
}
</style>
