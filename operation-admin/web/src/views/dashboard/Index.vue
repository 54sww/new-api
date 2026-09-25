<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'

const router = useRouter()
const appStore = useAppStore()

function goFinance() {
  appStore.switchSubsystem('finance')
  router.push('/finance/reconcile')
}

function goOps() {
  appStore.switchSubsystem('ops')
  router.push('/ops/overview')
}
function goSystem() {
  appStore.switchSubsystem('system')
  router.push('/system/users')
}
</script>

<template>
  <div class="page-container">
    <el-row :gutter="16">
      <el-col :xs="24" :sm="12" :md="8">
        <el-card shadow="hover" class="module-card" @click="goFinance">
          <div class="module-icon">
            <el-icon :size="28" color="#409eff"><Wallet /></el-icon>
          </div>
          <div class="module-title">财务管理</div>
          <div class="module-desc">渠道余额、充值登记、实时三方对账</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="8">
        <el-card shadow="hover" class="module-card" @click="goOps">
          <div class="module-icon">
            <el-icon :size="28" color="#67c23a"><Monitor /></el-icon>
          </div>
          <div class="module-title">运维管理</div>
          <div class="module-desc">运维模块占位，后续扩展</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="8">
        <el-card shadow="hover" class="module-card" @click="goSystem">
          <div class="module-icon">
            <el-icon :size="28" color="#e6a23c"><Lock /></el-icon>
          </div>
          <div class="module-title">系统管理</div>
          <div class="module-desc">用户、角色、菜单与按钮权限</div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped lang="scss">
.module-card {
  cursor: pointer;
  margin-bottom: 16px;
  transition: transform 0.15s;

  &:hover {
    transform: translateY(-2px);
  }
}

.module-icon {
  margin-bottom: 12px;
}

.module-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.module-desc {
  margin-top: 6px;
  font-size: 13px;
  color: $info-color;
}
</style>
