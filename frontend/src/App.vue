<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Activity, Check, CircleHelp, Eye, EyeOff, ExternalLink, Github, LogOut, RefreshCw, Settings as SettingsIcon, ShieldCheck, Wifi, X } from 'lucide-vue-next'
import { About, CheckUpdate, InstallUpdate, Login, Logout, MarkFrontendReady, Refresh, SaveSettings, Settings as LoadSettings, State, CloseToTray, ExitApp } from '../wailsjs/go/main/App'
import { BrowserOpenURL, EventsOn } from '../wailsjs/runtime/runtime'

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
  networks: string[]
  bytesIn4: number
  bytesOut4: number
  onlineCount: number
  terminals: string[]
  authCode: string
  authMessage: string
  dialCode: string
  dialMessage: string
  downloadRate: number
  uploadRate: number
}

type AboutInfo = { version: string; sha256: string; project: string; issues: string }
type UpdateInfo = { status: string; version: string; name: string; notes: string; url: string; publishedAt: string; assetUrl: string }

const state = ref<NetworkState>({ status: 'idle', message: '等待附近校园网络', ssid: '', interface: '', ip: '', mac: '', signal: '', account: '', lastChecked: '', networks: [], bytesIn4: 0, bytesOut4: 0, onlineCount: 0, terminals: [], authCode: '', authMessage: '', dialCode: '', dialMessage: '', downloadRate: 0, uploadRate: 0 })
const form = ref({ username: '', password: '', role: 'student', isp: 'cucc', remember: true, autoLogin: false, autoStart: false, exitBehavior: 'tray', skipExitPrompt: false })
const busy = ref(false)
const autoLoginPending = ref(false)
const error = ref('')
const showPassword = ref(false)
const view = ref<'login' | 'status' | 'traffic' | 'settings' | 'about'>('login')
const closeOpen = ref(false)
const closeRemember = ref(false)
const about = ref<AboutInfo>({ version: '0.1.0', sha256: '', project: 'https://github.com/XtSilan/hbasstuNet', issues: 'https://github.com/XtSilan/hbasstuNet/issues' })
const updateStatus = ref('')
const update = ref<UpdateInfo>({ status: '', version: '', name: '', notes: '', url: '', publishedAt: '', assetUrl: '' })
const installingUpdate = ref(false)
const rateHistory = ref<number[]>(Array.from({ length: 48 }, () => 0))
let rateTimer: number | undefined
let stopEvents: (() => void) | undefined

const connected = computed(() => state.value.status === 'connected')
const stateName = computed(() => ({ connected: '已连接', connecting: '正在认证', offline: '未连接校园网', error: '认证失败', idle: '等待连接' }[state.value.status] ?? '等待连接'))
const totalBytes = computed(() => state.value.bytesIn4 + state.value.bytesOut4)
function formatBytes(bytes: number) { if (!bytes) return '0 B'; const units = ['B', 'KB', 'MB', 'GB', 'TB']; const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1); return (bytes / Math.pow(1024, index)).toFixed(index ? 2 : 0) + ' ' + units[index] }
function formatRate(bytes: number) { return formatBytes(bytes) + '/s' }
const chartPoints = computed(() => { const values = rateHistory.value; const max = Math.max(...values, 1); return values.map((value, index) => `${(index / (values.length - 1)) * 100},${96 - (value / max) * 82}`).join(' ') })
function normalizeState(next: Partial<NetworkState> | null | undefined): NetworkState {
  const value = next ?? {}
  return {
    status: value.status || 'idle',
    message: value.message || '',
    ssid: value.ssid || '',
    interface: value.interface || '',
    ip: value.ip || '',
    mac: value.mac || '',
    signal: value.signal || '',
    account: value.account || '',
    lastChecked: value.lastChecked || '',
    networks: Array.isArray(value.networks) ? value.networks.filter(Boolean) : [],
    bytesIn4: Number.isFinite(Number(value.bytesIn4)) ? Number(value.bytesIn4) : 0,
    bytesOut4: Number.isFinite(Number(value.bytesOut4)) ? Number(value.bytesOut4) : 0,
    onlineCount: Number.isFinite(Number(value.onlineCount)) ? Number(value.onlineCount) : 0,
    terminals: Array.isArray(value.terminals) ? value.terminals.filter(Boolean) : [],
    authCode: value.authCode || '',
    authMessage: value.authMessage || '',
    dialCode: value.dialCode || '',
    dialMessage: value.dialMessage || '',
    downloadRate: Number.isFinite(Number(value.downloadRate)) ? Number(value.downloadRate) : 0,
    uploadRate: Number.isFinite(Number(value.uploadRate)) ? Number(value.uploadRate) : 0,
  }
}

onMounted(async () => {
  await MarkFrontendReady()
  stopEvents = EventsOn('state:changed', (next: NetworkState) => {
    const normalized = normalizeState(next)
    state.value = normalized
    if (normalized.status === 'connected' && autoLoginPending.value) {
      autoLoginPending.value = false
      if (view.value === 'login') view.value = 'status'
    }
    if (normalized.status === 'offline' && !normalized.ssid && view.value !== 'login') view.value = 'login'
  })
  EventsOn('close:requested', () => { closeOpen.value = true })
  EventsOn('navigate:settings', () => { void showSettings() })
  try {
    const saved = await LoadSettings()
    form.value = { ...form.value, ...saved, role: 'student' }
    state.value = normalizeState(await State())
    if (state.value.status === 'connected') view.value = 'status'
    autoLoginPending.value = form.value.autoLogin && !!form.value.username && !!form.value.password && (state.value.status === 'connecting')
    if (autoLoginPending.value) window.setTimeout(() => { autoLoginPending.value = false }, 60000)
  } catch (reason) {
    error.value = `初始化失败：${String(reason).replace(/^Error:\s*/, '')}`
  }
  rateTimer = window.setInterval(() => {
    const rate = Math.max(state.value.downloadRate, state.value.uploadRate)
    rateHistory.value = [...rateHistory.value.slice(1), rate]
  }, 1000)
})

onUnmounted(() => { stopEvents?.(); if (rateTimer) window.clearInterval(rateTimer) })

async function connect() {
  busy.value = true
  error.value = ''
  try {
    await Login(form.value.username.trim(), form.value.password, form.value.role, form.value.isp, form.value.remember)
    view.value = 'status'
  } catch (reason) {
    error.value = String(reason).replace(/^Error:\s*/, '')
  } finally {
    busy.value = false
  }
}

async function showSettings() {
  view.value = 'settings'
  try {
    const saved = await LoadSettings()
    form.value = { ...form.value, ...saved }
  } catch (reason) {
    error.value = String(reason).replace(/^Error:\s*/, '')
  }
}

async function disconnect() {
  busy.value = true
  error.value = ''
  try { await Logout(); form.value.role = 'student'; view.value = 'login' } catch (reason) { error.value = String(reason) } finally { busy.value = false }
}

async function refresh() {
  try { state.value = normalizeState(await Refresh()) } catch (reason) { error.value = String(reason).replace(/^Error:\s*/, '') }
}

async function savePreferences() {
  try { await SaveSettings(form.value) } catch (reason) { error.value = String(reason) }
}

async function closeToTray() {
  closeOpen.value = false
  try { await CloseToTray(closeRemember.value) } catch (reason) { error.value = String(reason) }
}

async function exitApp() {
  closeOpen.value = false
  try { await ExitApp(closeRemember.value) } catch (reason) { error.value = String(reason) }
}

async function showAbout() {
  view.value = 'about'
  try { about.value = await About() } catch (reason) { error.value = String(reason) }
}

function openProject() { BrowserOpenURL(about.value.project) }
function openIssues() { BrowserOpenURL(about.value.issues) }
async function checkUpdates() {
  updateStatus.value = '正在检查…'
  try {
    update.value = await CheckUpdate()
    updateStatus.value = update.value.version ? `最新版本 ${update.value.version}` : update.value.status
  } catch (reason) {
    updateStatus.value = String(reason).replace(/^Error:\s*/, '')
  }
}

async function installUpdate() {
  if (!update.value.assetUrl) return
  installingUpdate.value = true
  updateStatus.value = '正在下载更新…'
  try {
    await InstallUpdate(update.value.assetUrl)
  } catch (reason) {
    installingUpdate.value = false
    updateStatus.value = String(reason).replace(/^Error:\s*/, '')
  }
}
</script>

<template>
  <main class="app-shell">
    <header class="titlebar">
      <div class="app-identity"><img src="./assets/images/campus.svg" alt="" /><span>理工校园网登录器</span></div>
    </header>

    <!-- Keep a concrete page mounted for every view value. In particular, a state
         event can report connected while the user is returning from settings.
         The old `&& !connected` guard then made both login and status sections
         disappear for one render, leaving a black WebView surface. -->
    <section v-if="view === 'login'" class="login-view">
      <aside class="network-side">
        <div class="section-label">校园网络</div>
        <div class="wifi-display">
          <span class="wifi-ring ring-one"></span><span class="wifi-ring ring-two"></span><span class="wifi-ring ring-three"></span>
          <Wifi :size="54" stroke-width="1.4" />
        </div>
        <div class="nearby-networks">
          <div class="nearby-heading"><strong>附近网络</strong><button class="scan-refresh" type="button" :disabled="busy" @click="refresh"><RefreshCw :size="13" />刷新</button></div>
          <button v-for="network in state.networks" :key="network" class="network-option" :class="{ selected: form.role === (network.toLowerCase().startsWith('tercher') || network.toLowerCase().startsWith('teacher') ? 'teacher' : 'student') }" @click="form.role = network.toLowerCase().startsWith('tercher') || network.toLowerCase().startsWith('teacher') ? 'teacher' : 'student'">
            <span class="network-light"></span><Wifi :size="17" /><span>{{ network }}</span><small>可用</small>
          </button>
          <div v-if="!state.networks.length" class="scan-placeholder"><span class="scan-pulse"></span><span>扫描附近 Wi-Fi</span></div>
        </div>
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
            <label class="check-row"><input v-model="form.autoLogin" type="checkbox" @change="savePreferences" /><span><Check :size="11" /></span>自动登录</label>
          </div>
          <p v-if="error" class="error-message">{{ error }}</p>
          <button class="primary-button" type="submit" :disabled="busy || autoLoginPending || !form.username || !form.password">{{ busy ? '正在认证…' : autoLoginPending ? '自动登录中…' : '连接校园网' }}</button>
        </form>
        <div class="login-status"><span :class="['state-dot', state.status]"></span><div><strong>{{ stateName }}</strong><span>{{ state.ssid || state.message }}</span></div></div>
      </section>
    </section>

    <section v-else-if="view === 'status' || view === 'traffic'" class="status-view">
      <nav class="sidebar"><button :class="{ active: view === 'status' }" title="网络状态" @click="view = 'status'"><Wifi :size="19" /></button><button :class="{ active: view === 'traffic' }" title="流量面板" @click="view = 'traffic'"><Activity :size="19" /></button><button title="设置" @click="showSettings"><SettingsIcon :size="19" /></button><button title="关于" @click="showAbout"><CircleHelp :size="19" /></button></nav>
      <section class="status-content">
        <div class="status-heading"><div><span class="section-label">{{ view === 'traffic' ? '流量概览' : '连接状态' }}</span><h1>{{ view === 'traffic' ? '网络流量' : '校园网络' }}</h1></div><button class="secondary-button" @click="refresh"><RefreshCw :size="15" />刷新</button></div>
        <template v-if="view === 'traffic'"><section class="traffic-summary"><div class="traffic-total"><span>网络速度</span><strong>↑ {{ formatRate(state.uploadRate) }}　↓ {{ formatRate(state.downloadRate) }}</strong></div><div class="traffic-line-chart"><svg viewBox="0 0 100 100" preserveAspectRatio="none"><polyline :points="chartPoints" /></svg></div><div class="traffic-counters"><span>下行总量 <strong>{{ formatBytes(state.bytesIn4) }}</strong></span><span>上行总量 <strong>{{ formatBytes(state.bytesOut4) }}</strong></span></div></section><section class="details traffic-details"><h3>在线信息</h3><dl><div><dt>在线设备</dt><dd>{{ state.onlineCount || 0 }}</dd></div><div><dt>物理地址</dt><dd>{{ state.mac || '—' }}</dd></div><div><dt>认证状态</dt><dd>{{ state.authCode || '—' }}</dd></div><div><dt>拨号状态</dt><dd>{{ state.dialCode || '—' }}</dd></div><div><dt>认证提示</dt><dd>{{ state.authMessage || state.message || '—' }}</dd></div><div><dt>终端列表</dt><dd>{{ state.terminals.length ? state.terminals.join('、') : '—' }}</dd></div></dl></section></template>
        <template v-else>
        <div class="connection-status"><div class="connected-icon"><Wifi :size="34" /></div><div><span class="connected-label"><i></i>连接正常</span><h2>{{ state.ssid }}</h2><p>{{ state.message }}</p></div><button class="danger-button" :disabled="busy" @click="disconnect"><LogOut :size="15" />断开连接</button></div>
        <section class="details"><h3>网络信息</h3><dl><div><dt>认证账号</dt><dd>{{ state.account || '—' }}</dd></div><div><dt>无线网络</dt><dd>{{ state.ssid || '—' }}</dd></div><div><dt>IPv4 地址</dt><dd>{{ state.ip || '—' }}</dd></div><div><dt>网络接口</dt><dd>{{ state.interface || '—' }}</dd></div><div><dt>物理地址</dt><dd>{{ state.mac || '—' }}</dd></div><div><dt>信号强度</dt><dd>{{ state.signal || '—' }}</dd></div></dl></section>
        <footer class="status-footer"><span><i></i>后台状态检查已启用</span><span>上次检查 {{ state.lastChecked || '刚刚' }}</span></footer></template>
      </section>
    </section>

    <section v-if="view === 'settings'" class="status-view">
      <nav class="sidebar"><button title="网络状态" @click="view = 'status'"><Wifi :size="19" /></button><button title="流量面板" @click="view = 'traffic'"><Activity :size="19" /></button><button class="active" title="设置"><SettingsIcon :size="19" /></button><button title="关于" @click="showAbout"><CircleHelp :size="19" /></button></nav>
      <section class="status-content">
      <header class="about-header"><div><span class="section-label">设置</span><h1>应用设置</h1></div></header>
      <section class="settings-panel">
        <div class="settings-row"><div><strong>保存密码</strong><span>使用 Windows 用户凭据保护保存的密码</span></div><label class="switch"><input v-model="form.remember" type="checkbox" @change="savePreferences" /><span></span></label></div>
        <div class="settings-row"><div><strong>自动登录</strong><span>启动后发现校园 Wi-Fi 后使用已保存账号认证</span></div><label class="switch"><input v-model="form.autoLogin" type="checkbox" @change="savePreferences" /><span></span></label></div>
        <div class="settings-row"><div><strong>开机自启动</strong><span>登录 Windows 后在后台运行 hbasstuNet</span></div><label class="switch"><input v-model="form.autoStart" type="checkbox" @change="savePreferences" /><span></span></label></div>
        <div class="settings-row"><div><strong>关闭窗口时</strong><span>选择点击右上角关闭按钮后的行为</span></div><select v-model="form.exitBehavior" @change="savePreferences"><option value="tray">最小化到系统托盘</option><option value="exit">直接退出</option></select></div>
      </section>
      </section>
    </section>

    <section v-if="view === 'about'" class="status-view">
      <nav class="sidebar"><button title="网络状态" @click="view = 'status'"><Wifi :size="19" /></button><button title="流量面板" @click="view = 'traffic'"><Activity :size="19" /></button><button title="设置" @click="showSettings"><SettingsIcon :size="19" /></button><button class="active" title="关于"><CircleHelp :size="19" /></button></nav>
      <section class="status-content">
      <header class="about-header"><div><span class="section-label">关于</span><h1>关于 hbasstuNet</h1></div></header>
      <section class="about-panel">
        <div class="about-brand"><img src="./assets/images/campus.svg" alt="" /><div><strong>hbasstuNet</strong><span>湖北文理学院理工学院校园网登录器</span></div></div>
        <dl class="about-details"><div><dt>版本</dt><dd>v{{ about.version }}</dd></div><div><dt>SHA-256</dt><dd class="hash">{{ about.sha256 || '正在计算…' }}</dd></div><div><dt>项目地址</dt><dd>{{ about.project.replace('https://', '') }}</dd></div></dl>
        <div class="about-actions"><button class="secondary-button" @click="openProject"><Github :size="15" />打开项目主页</button><button class="secondary-button" @click="openIssues"><ExternalLink :size="15" />反馈问题</button></div>
      </section>
      <section class="update-panel"><div class="update-heading"><div><span class="section-label">更新</span><h2>GitHub Release 更新</h2></div><span class="current-version">当前版本 v{{ about.version }}</span></div><p>获取最新版本和发布说明。</p><div v-if="updateStatus" class="release-notes"><strong>{{ update.name || update.version || update.status }}</strong><span>{{ update.notes || '暂无发布说明' }}</span></div><div class="update-actions"><button class="update-button" @click="checkUpdates" :disabled="installingUpdate"><RefreshCw :size="15" />检查更新</button><button v-if="update.assetUrl" class="update-button" @click="installUpdate" :disabled="installingUpdate"><ExternalLink :size="15" />{{ installingUpdate ? '正在安装…' : '下载并安装' }}</button><button v-if="update.url" class="secondary-button" @click="BrowserOpenURL(update.url)"><ExternalLink :size="15" />查看发布页</button><span v-if="updateStatus" class="update-status">{{ updateStatus }}</span></div></section>
      </section>
    </section>

    <div v-if="closeOpen" class="modal-backdrop"><section class="close-dialog"><button class="dialog-close" title="取消" @click="closeOpen = false"><X :size="16" /></button><h2>关闭 hbasstuNet</h2><p>请选择关闭窗口后的操作。</p><div class="close-actions"><button class="secondary-button" @click="closeToTray">最小化到托盘</button><button class="danger-button" @click="exitApp">直接退出</button></div><label class="check-row"><input v-model="closeRemember" type="checkbox" /><span><Check :size="11" /></span>下次不再提示</label></section></div>
  </main>
</template>
