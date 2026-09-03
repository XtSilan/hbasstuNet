import { createApp } from 'vue'
import App from './App.vue'
import './style.css'

try {
  createApp(App).mount('#app')
} catch (reason) {
  const root = document.querySelector('#app')
  if (root) root.textContent = `hbasstuNet 前端加载失败：${String(reason)}`
  console.error(reason)
}
