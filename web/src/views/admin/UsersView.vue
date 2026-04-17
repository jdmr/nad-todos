<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listUsers, setUserRole } from '@/api/admin'
import type { AdminUser } from '@/api/admin'
import type { Role } from '@/api/auth'

const users = ref<AdminUser[]>([])
const error = ref<string | null>(null)

async function load() {
  try {
    users.value = await listUsers()
  } catch {
    error.value = 'Failed to load users'
  }
}
onMounted(load)

async function changeRole(u: AdminUser, role: Role) {
  if (u.role === role) return
  try {
    await setUserRole(u.id, role)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to update role'
  }
}
</script>

<template>
  <main class="page">
    <h1 class="title">Users</h1>
    <div class="rule" />

    <p v-if="error" class="error">{{ error }}</p>

    <table v-if="users.length" class="table">
      <thead>
        <tr><th>Email</th><th>Name</th><th>Role</th><th></th></tr>
      </thead>
      <tbody>
        <tr v-for="u in users" :key="u.id">
          <td>{{ u.email }}</td>
          <td>{{ u.name }}</td>
          <td>{{ u.role }}</td>
          <td>
            <button v-if="u.role === 'user'" class="link" @click="changeRole(u, 'admin')">make admin</button>
            <button v-else class="link" @click="changeRole(u, 'user')">demote</button>
          </td>
        </tr>
      </tbody>
    </table>
  </main>
</template>

<style scoped>
.page { padding-top: 4rem; max-width: 48rem; }
.title { font-family: var(--font-display); font-size: clamp(2rem, 5vw, 2.75rem); font-weight: 400; letter-spacing: -0.02em; }
.rule { width: 2.5rem; height: 3px; background: var(--color-accent); margin: 0.875rem 0 2rem; border-radius: 2px; }
.table { width: 100%; border-collapse: collapse; }
.table th, .table td { text-align: left; padding: 0.5rem 0.75rem; border-bottom: 1px solid var(--color-border, #eee); }
.table th { font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--color-text-muted); }
.link { background: none; border: none; color: var(--color-accent); cursor: pointer; font-size: 0.85rem; }
.error { color: var(--color-error-text, #c00); background: var(--color-error-bg, #fee); padding: 0.5rem 0.75rem; border-radius: 6px; margin-bottom: 1rem; }
</style>
