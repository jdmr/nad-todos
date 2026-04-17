<template>
  <div class="app">
    <nav v-if="auth.isAuthenticated" class="nav">
      <RouterLink to="/" class="brand">Todos</RouterLink>
      <div class="links">
        <RouterLink to="/account">Account</RouterLink>
        <template v-if="auth.isAdmin">
          <RouterLink to="/admin/users">Users</RouterLink>
          <RouterLink to="/admin/invitations">Invitations</RouterLink>
        </template>
        <button class="logout" @click="onLogout">Sign out</button>
      </div>
    </nav>
    <RouterView />
  </div>
</template>

<script setup lang="ts">
import { RouterView, RouterLink, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

async function onLogout() {
  await auth.logout()
  router.push({ name: 'login' })
}
</script>

<style scoped>
.app { max-width: 64rem; margin: 0 auto; padding: 0 1.5rem; }
.nav {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.25rem 0;
  border-bottom: 1px solid var(--color-border, #eee);
}
.brand { font-family: var(--font-display); font-size: 1.25rem; text-decoration: none; color: var(--color-text); }
.links { display: flex; gap: 1.25rem; align-items: center; font-size: 0.875rem; }
.links a { color: var(--color-text-muted); text-decoration: none; }
.links a.router-link-active { color: var(--color-text); }
.logout { background: none; border: none; color: var(--color-text-muted); cursor: pointer; font-size: 0.875rem; }
</style>
