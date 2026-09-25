<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const form = reactive({
  username: '',
  password: '',
  pat: '',
  code: '',
})
const flowToken = ref('')
const loading = ref(false)

async function handleLogin() {
  loading.value = true
  try {
    if (flowToken.value) {
      await userStore.login2fa(flowToken.value, form.code)
      ElMessage.success('登录成功')
      router.push((route.query.redirect as string) || '/')
      return
    }
    const data = await userStore.login({
      username: form.username,
      password: form.password,
      pat: form.pat,
    })
    if (data.require_2fa) {
      flowToken.value = data.flow_token || ''
      ElMessage.info('请输入二次验证码')
      return
    }
    ElMessage.success('登录成功')
    router.push((route.query.redirect as string) || '/')
  } catch {
    // 错误已在拦截器提示
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <h1 class="login-title">Operation Admin</h1>
      <p class="login-subtitle">运营管理后台 · 本地账号或 new-api Root</p>

      <el-form :model="form" size="large" @keyup.enter="handleLogin">
        <template v-if="!flowToken">
          <el-form-item>
            <el-input v-model="form.username" placeholder="用户名" :prefix-icon="'User'" />
          </el-form-item>
          <el-form-item>
            <el-input
              v-model="form.password"
              type="password"
              placeholder="密码"
              show-password
              :prefix-icon="'Lock'"
            />
          </el-form-item>
          <el-form-item>
            <el-input
              v-model="form.pat"
              type="password"
              placeholder="或系统访问令牌（PAT）"
              show-password
              :prefix-icon="'Key'"
            />
          </el-form-item>
        </template>
        <el-form-item v-else>
          <el-input v-model="form.code" placeholder="二次验证码" :prefix-icon="'Key'" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" style="width: 100%" @click="handleLogin">
            登 录
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<style scoped lang="scss">
.login-page {
  width: 100vw;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #409eff, #304156);
}

.login-card {
  width: 400px;
  padding: 40px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);

  .login-title {
    text-align: center;
    font-size: 28px;
    color: #303133;
    margin-bottom: 4px;
  }

  .login-subtitle {
    text-align: center;
    font-size: 14px;
    color: #909399;
    margin-bottom: 32px;
  }
}
</style>
