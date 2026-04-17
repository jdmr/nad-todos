import 'virtual:uno.css'
import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'

import App from './App.vue'
import router from './router'
import { setOn401Handler } from './api/client'
import { useAuthStore } from './stores/auth'

const app = createApp(App)

const pinia = createPinia()
pinia.use(piniaPluginPersistedstate)
app.use(pinia)
app.use(router)

const auth = useAuthStore()
setOn401Handler(() => {
  auth.clearUser()
  if (router.currentRoute.value.meta.requiresAuth) {
    router.push({ name: 'login' })
  }
})

app.mount('#app')
