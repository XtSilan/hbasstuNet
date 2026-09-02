<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ArrowLeftRight, Check, Eye, EyeOff, LogOut, RefreshCw, Settings as SettingsIcon, ShieldCheck, Wifi } from 'lucide-vue-next'
import { Login, Logout, Refresh, SaveSettings, Settings, State } from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'

type NetworkState = {
  status: string
  message: string
  ssid: string
  interface: string
  ip: string
  mac: string
  signal: string
  account: string
  lastChecked: string
}

const state = ref<NetworkState>({ status: 'idle', message: '等待连接', ssid: '', interface: '', ip: '', mac: '', signal: '', account: '', lastChecked: '' })
const form = ref({ username: '', password: '', role: 'student', isp: 'cucc', remember: true, autoStart: false })
const busy = ref(false)
const error = ref('')
const showPassword = ref(false)
let stopEvents: (() => void) | undefined

const connected = computed(() => state.value.status === 'connected')
const roleName = computed(() => form.value.role === 'student' ? 'Student-XYW' : 'Tercher-XYW')
const stateName = computed(() => ({ connected: '已连接', connecting: '正在认证', offline: '未连接校园网', error: '认证失败', idle: '等待连接' }[state.value.status] ?? '等待连接'))

onMounted(async () => {
  const saved = await Settings()
  form.value = { ...form.value, ...saved }
  state.value = await State()
  stopEvents = EventsOn('state:changed', (next: NetworkState) => { state.value = next })
})

onUnmounted(() => stopEvents?.())

async function connect() {
  busy.value = true
  error.value = ''
  try {
    await Login(form.value.username.trim(), form.value.password, form.value.role, form.value.isp, form.value.remember)
  } catch (reason) {
    error.value = String(reason).replace(/^Error:\s*/, '')
  } finally {
    busy.value = false
  }
}

async function disconnect() {
  busy.value = true
  error.value = ''
  try { await Logout() } catch (reason) { error.value = String(reason) } finally { busy.value = false }
}

async function refresh() {
  state.value = await Refresh()
}

async function savePreferences() {
  try { await SaveSettings(form.value) } catch (reason) { error.value = String(reason) }
}
</script>

<template>
  <main class="app-shell">
    <header class="titlebar">
      <div class="app-identity"><Wifi :size="18" /><span>理工校园网登录器</span></div>
      <div class="title-actions"><button class="icon-button" title="刷新网络" @click="refresh"><RefreshCw :size="16" /></button><span>hbasstuNet</span></div>
    </header>

    <section v-if="!connected" class="login-view">
      <aside class="network-side">
        <div class="section-label">校园网络</div>
        <div class="wifi-display">
          <span class="wifi-ring ring-one"></span><span class="wifi-ring ring-two"></span><span class="wifi-ring ring-three"></span>
          <Wifi :size="54" stroke-width="1.4" />
        </div>
        <div class="network-selector">
          <button title="切换账号类型" @click="form.role = form.role === 'student' ? 'teacher' : 'student'"><ArrowLeftRight :size="16" /></button>
          <div><strong>{{ roleName }}</strong><span>{{ form.role === 'student' ? '学生网络' : '教师网络' }}</span></div>
        </div>
        <div class="network-state"><span :class="['state-dot', state.status]"></span><div><strong>{{ stateName }}</strong><span>{{ state.ssid || state.message }}</span></div></div>
      </aside>

      <section class="login-form">
        <div class="form-heading"><ShieldCheck :size="22" /><h1>校园网认证</h1></div>
        <div class="role-tabs">
          <button :class="{ active: form.role === 'student' }" @click="form.role = 'student'">学生登录</button>
          <button :class="{ active: form.role === 'teacher' }" @click="form.role = 'teacher'">教师登录</button>
        </div>
        <form @submit.prevent="connect">
          <label><span>校园网账号</span><input v-model="form.username" autocomplete="username" placeholder="请输入账号" /></label>
          <label><span>密码</span><div class="password-input"><input v-model="form.password" :type="showPassword ? 'text' : 'password'" autocomplete="current-password" placeholder="请输入密码" /><button type="button" :title="showPassword ? '隐藏密码' : '显示密码'" @click="showPassword = !showPassword"><EyeOff v-if="showPassword" :size="16" /><Eye v-else :size="16" /></button></div></label>
          <label><span>运营商</span><select v-model="form.isp"><option value="cucc">中国联通</option><option value="cmcc">中国移动</option><option value="telecom">中国电信</option></select></label>
          <div class="preferences">
            <label class="check-row"><input v-model="form.remember" type="checkbox" @change="savePreferences" /><span><Check :size="11" /></span>保存密码</label>
            <label class="check-row"><input v-model="form.autoStart" type="checkbox" @change="savePreferences" /><span><Check :size="11" /></span>开机启动</label>
          </div>
          <p v-if="error" class="error-message">{{ error }}</p>
          <button class="primary-button" type="submit" :disabled="busy || !form.username || !form.password">{{ busy ? '正在认证…' : '连接校园网' }}</button>
        </form>
        <p class="form-note">仅在识别到 Student-XYW 或 Tercher-XYW 后发起认证</p>
      </section>
    </section>

    <section v-else class="status-view">
      <nav class="sidebar"><button class="active" title="网络状态"><Wifi :size="19" /></button><button title="设置"><SettingsIcon :size="19" /></button></nav>
      <section class="status-content">
        <div class="status-heading"><div><span class="section-label">连接状态</span><h1>校园网络</h1></div><button class="secondary-button" @click="refresh"><RefreshCw :size="15" />刷新</button></div>
        <div class="connection-status"><div class="connected-icon"><Wifi :size="34" /></div><div><span class="connected-label"><i></i>连接正常</span><h2>{{ state.ssid }}</h2><p>{{ state.message }}</p></div><button class="danger-button" :disabled="busy" @click="disconnect"><LogOut :size="15" />断开连接</button></div>
        <section class="details"><h3>网络信息</h3><dl><div><dt>认证账号</dt><dd>{{ state.account || '—' }}</dd></div><div><dt>无线网络</dt><dd>{{ state.ssid || '—' }}</dd></div><div><dt>IPv4 地址</dt><dd>{{ state.ip || '—' }}</dd></div><div><dt>网络接口</dt><dd>{{ state.interface || '—' }}</dd></div><div><dt>物理地址</dt><dd>{{ state.mac || '—' }}</dd></div><div><dt>信号强度</dt><dd>{{ state.signal || '—' }}</dd></div></dl></section>
        <footer class="status-footer"><span><i></i>后台状态检查已启用</span><span>上次检查 {{ state.lastChecked || '刚刚' }}</span></footer>
      </section>
    </section>
  </main>
</template>
