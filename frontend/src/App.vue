<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ArrowLeft, Check, CircleHelp, Eye, EyeOff, ExternalLink, Github, LogOut, RefreshCw, Settings as SettingsIcon, ShieldCheck, Wifi } from 'lucide-vue-next'
import { About, CheckUpdate, Login, Logout, MarkFrontendReady, Refresh, SaveSettings, Settings as LoadSettings, State, CloseToTray, ExitApp } from '../wailsjs/go/main/App'
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
}

type AboutInfo = { version: string; sha256: string; project: string; issues: string }
type UpdateInfo = { status: string; version: string; name: string; notes: string; url: string; publishedAt: string }

const state = ref<NetworkState>({ status: 'idle', message: '等待附近校园网络', ssid: '', interface: '', ip: '', mac: '', signal: '', account: '', lastChecked: '', networks: [] })
const form = ref({ username: '', password: '', role: 'student', isp: 'cucc', remember: true, autoLogin: false, exitBehavior: 'tray', skipExitPrompt: false })
const busy = ref(false)
const error = ref('')
const showPassword = ref(false)
const aboutOpen = ref(false)
const settingsOpen = ref(false)
const closeOpen = ref(false)
const closeRemember = ref(false)
const about = ref<AboutInfo>({ version: '0.1.0', sha256: '', project: 'https://github.com/XtSilan/hbasstuNet', issues: 'https://github.com/XtSilan/hbasstuNet/issues' })
const updateStatus = ref('')
const update = ref<UpdateInfo>({ status: '', version: '', name: '', notes: '', url: '', publishedAt: '' })
let stopEvents: (() => void) | undefined

const connected = computed(() => state.value.status === 'connected')
const stateName = computed(() => ({ connected: '已连接', connecting: '正在认证', offline: '未连接校园网', error: '认证失败', idle: '等待连接' }[state.value.status] ?? '等待连接'))

onMounted(async () => {
  await MarkFrontendReady()
  stopEvents = EventsOn('state:changed', (next: NetworkState) => { state.value = next })
  EventsOn('close:requested', () => { closeOpen.value = true })
  EventsOn('navigate:settings', () => { settingsOpen.value = true; aboutOpen.value = false })
  try {
    const saved = await LoadSettings()
    form.value = { ...form.value, ...saved }
    state.value = await State()
  } catch (reason) {
    error.value = `初始化失败：${String(reason).replace(/^Error:\s*/, '')}`
  }
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

async function closeToTray() {
  closeOpen.value = false
  try { await CloseToTray(closeRemember.value) } catch (reason) { error.value = String(reason) }
}

async function exitApp() {
  closeOpen.value = false
  try { await ExitApp(closeRemember.value) } catch (reason) { error.value = String(reason) }
}

async function showAbout() {
  aboutOpen.value = true
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
</script>

<template>
  <main class="app-shell">
    <header class="titlebar">
      <div class="app-identity"><img src="./assets/images/campus.svg" alt="" /><span>理工校园网登录器</span></div>
      <div class="title-actions"><button class="icon-button" title="设置" @click="settingsOpen = true"><SettingsIcon :size="16" /></button><button class="icon-button" title="扫描附近网络" @click="refresh"><RefreshCw :size="16" /></button></div>
    </header>

    <section v-if="!connected && !settingsOpen && !aboutOpen" class="login-view">
      <aside class="network-side">
        <div class="section-label">校园网络</div>
        <div class="wifi-display">
          <span class="wifi-ring ring-one"></span><span class="wifi-ring ring-two"></span><span class="wifi-ring ring-three"></span>
          <Wifi :size="54" stroke-width="1.4" />
        </div>
        <div class="nearby-networks">
          <div class="nearby-heading"><strong>附近网络</strong><span>{{ state.networks.length ? '发现可用校园网络' : '正在扫描' }}</span></div>
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
          </div>
          <p v-if="error" class="error-message">{{ error }}</p>
          <button class="primary-button" type="submit" :disabled="busy || !form.username || !form.password">{{ busy ? '正在认证…' : '连接校园网' }}</button>
        </form>
        <div class="login-status"><span :class="['state-dot', state.status]"></span><div><strong>{{ stateName }}</strong><span>{{ state.ssid || state.message }}</span></div></div>
      </section>
    </section>

    <section v-else-if="!aboutOpen && !settingsOpen" class="status-view">
      <nav class="sidebar"><button class="active" title="网络状态"><Wifi :size="19" /></button><button title="设置" @click="settingsOpen = true"><SettingsIcon :size="19" /></button><button title="关于" @click="showAbout"><CircleHelp :size="19" /></button></nav>
      <section class="status-content">
        <div class="status-heading"><div><span class="section-label">连接状态</span><h1>校园网络</h1></div><button class="secondary-button" @click="refresh"><RefreshCw :size="15" />刷新</button></div>
        <div class="connection-status"><div class="connected-icon"><Wifi :size="34" /></div><div><span class="connected-label"><i></i>连接正常</span><h2>{{ state.ssid }}</h2><p>{{ state.message }}</p></div><button class="danger-button" :disabled="busy" @click="disconnect"><LogOut :size="15" />断开连接</button></div>
        <section class="details"><h3>网络信息</h3><dl><div><dt>认证账号</dt><dd>{{ state.account || '—' }}</dd></div><div><dt>无线网络</dt><dd>{{ state.ssid || '—' }}</dd></div><div><dt>IPv4 地址</dt><dd>{{ state.ip || '—' }}</dd></div><div><dt>网络接口</dt><dd>{{ state.interface || '—' }}</dd></div><div><dt>物理地址</dt><dd>{{ state.mac || '—' }}</dd></div><div><dt>信号强度</dt><dd>{{ state.signal || '—' }}</dd></div></dl></section>
        <footer class="status-footer"><span><i></i>后台状态检查已启用</span><span>上次检查 {{ state.lastChecked || '刚刚' }}</span></footer>
      </section>
    </section>

    <section v-if="settingsOpen && !aboutOpen" class="settings-view">
      <header class="about-header"><button class="icon-button" title="返回网络状态" @click="settingsOpen = false"><ArrowLeft :size="16" /></button><div><span class="section-label">设置</span><h1>应用设置</h1></div></header>
      <section class="settings-panel">
        <div class="settings-row"><div><strong>自动登录</strong><span>启动后检测校园 Wi-Fi 并使用已保存账号认证</span></div><label class="switch"><input v-model="form.autoLogin" type="checkbox" @change="savePreferences" /><span></span></label></div>
        <div class="settings-row"><div><strong>保存密码</strong><span>使用 Windows 用户凭据保护保存的密码</span></div><label class="switch"><input v-model="form.remember" type="checkbox" @change="savePreferences" /><span></span></label></div>
        <div class="settings-row"><div><strong>关闭窗口时</strong><span>选择点击右上角关闭按钮后的行为</span></div><select v-model="form.exitBehavior" @change="savePreferences"><option value="tray">最小化到系统托盘</option><option value="exit">直接退出</option></select></div>
        <div class="settings-row"><div><strong>下次不再询问</strong><span>关闭窗口时直接使用上面的选择</span></div><label class="switch"><input v-model="form.skipExitPrompt" type="checkbox" @change="savePreferences" /><span></span></label></div>
      </section>
    </section>

    <section v-if="aboutOpen" class="about-view">
      <header class="about-header"><button class="icon-button" title="返回网络状态" @click="aboutOpen = false"><ArrowLeft :size="16" /></button><div><span class="section-label">关于</span><h1>关于 hbasstuNet</h1></div></header>
      <section class="about-panel">
        <div class="about-brand"><img src="./assets/images/campus.svg" alt="" /><div><strong>hbasstuNet</strong><span>湖北文理学院理工学院校园网登录器</span></div></div>
        <dl class="about-details"><div><dt>版本</dt><dd>v{{ about.version }}</dd></div><div><dt>SHA-256</dt><dd class="hash">{{ about.sha256 || '正在计算…' }}</dd></div><div><dt>项目地址</dt><dd>{{ about.project.replace('https://', '') }}</dd></div></dl>
        <div class="about-actions"><button class="secondary-button" @click="openProject"><Github :size="15" />打开项目主页</button><button class="secondary-button" @click="openIssues"><ExternalLink :size="15" />反馈问题</button></div>
      </section>
      <section class="update-panel"><div class="update-heading"><div><span class="section-label">更新</span><h2>GitHub Release 更新</h2></div><span class="current-version">当前版本 v{{ about.version }}</span></div><p>获取最新版本和发布说明。</p><div v-if="updateStatus" class="release-notes"><strong>{{ update.name || update.version || update.status }}</strong><span>{{ update.notes || '暂无发布说明' }}</span></div><div class="update-actions"><button class="update-button" @click="checkUpdates"><RefreshCw :size="15" />检查更新</button><button v-if="update.url" class="secondary-button" @click="BrowserOpenURL(update.url)"><ExternalLink :size="15" />查看发布页</button><span v-if="updateStatus" class="update-status">{{ updateStatus }}</span></div></section>
    </section>

    <div v-if="closeOpen" class="modal-backdrop"><section class="close-dialog"><h2>关闭 hbasstuNet</h2><p>请选择关闭窗口后的操作。</p><div class="close-actions"><button class="secondary-button" @click="closeToTray">最小化到托盘</button><button class="danger-button" @click="exitApp">直接退出</button></div><label class="check-row"><input v-model="closeRemember" type="checkbox" /><span><Check :size="11" /></span>下次不再提示</label></section></div>
  </main>
</template>
