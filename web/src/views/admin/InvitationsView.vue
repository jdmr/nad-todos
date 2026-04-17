<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  listInvitations,
  createInvitation,
  revokeInvitation,
} from '@/api/admin'
import type { AdminInvitation } from '@/api/admin'
import type { Role } from '@/api/auth'

const invitations = ref<AdminInvitation[]>([])
const error = ref<string | null>(null)
const newEmail = ref('')
const newRole = ref<Role>('user')
const lastLink = ref<string | null>(null)

async function load() {
  try {
    invitations.value = await listInvitations()
  } catch {
    error.value = 'Failed to load invitations'
  }
}
onMounted(load)

async function create() {
  if (!newEmail.value) return
  try {
    const inv = await createInvitation(newEmail.value, newRole.value)
    lastLink.value = `${window.location.origin}/register?token=${encodeURIComponent(inv.token)}`
    newEmail.value = ''
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to create invitation'
  }
}

async function revoke(id: string) {
  await revokeInvitation(id)
  await load()
}

function copy(text: string) {
  navigator.clipboard.writeText(text)
}

function inviteLink(token: string) {
  return `${window.location.origin}/register?token=${encodeURIComponent(token)}`
}
</script>

<template>
  <main class="page">
    <h1 class="title">Invitations</h1>
    <div class="rule" />

    <p v-if="error" class="error">{{ error }}</p>

    <form class="form" @submit.prevent="create">
      <input v-model="newEmail" type="email" required class="input" placeholder="user@example.com" />
      <select v-model="newRole" class="input">
        <option value="user">user</option>
        <option value="admin">admin</option>
      </select>
      <button type="submit" class="button">Invite</button>
    </form>

    <p v-if="lastLink" class="link-note">
      New invitation link
      <button class="link" @click="copy(lastLink!)">copy</button>
      <code class="code">{{ lastLink }}</code>
    </p>

    <table v-if="invitations.length" class="table">
      <thead>
        <tr><th>Email</th><th>Role</th><th>Status</th><th>Expires</th><th></th></tr>
      </thead>
      <tbody>
        <tr v-for="i in invitations" :key="i.id">
          <td>{{ i.email }}</td>
          <td>{{ i.default_role }}</td>
          <td>{{ i.status }}</td>
          <td>{{ new Date(i.expires_at).toLocaleString() }}</td>
          <td>
            <button v-if="i.status === 'pending'" class="link" @click="copy(inviteLink(i.token))">copy link</button>
            <button v-if="i.status === 'pending'" class="link danger" @click="revoke(i.id)">revoke</button>
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
.form { display: flex; gap: 0.5rem; margin-bottom: 1.5rem; }
.input { padding: 0.5rem 0.75rem; border: 1px solid var(--color-border, #ddd); border-radius: 6px; flex: 1; }
.button { padding: 0.5rem 0.875rem; background: var(--color-text); color: var(--color-bg); border: none; border-radius: 6px; cursor: pointer; }
.table { width: 100%; border-collapse: collapse; }
.table th, .table td { text-align: left; padding: 0.5rem 0.75rem; border-bottom: 1px solid var(--color-border, #eee); font-size: 0.875rem; }
.table th { font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--color-text-muted); }
.link { background: none; border: none; color: var(--color-accent); cursor: pointer; font-size: 0.85rem; margin-right: 0.5rem; }
.link.danger { color: var(--color-error-text, #c00); }
.link-note { display: flex; align-items: center; gap: 0.5rem; margin-bottom: 1rem; font-size: 0.85rem; }
.code { background: var(--color-bg-muted, #f5f5f5); padding: 0.125rem 0.375rem; border-radius: 4px; font-size: 0.75rem; word-break: break-all; }
.error { color: var(--color-error-text, #c00); background: var(--color-error-bg, #fee); padding: 0.5rem 0.75rem; border-radius: 6px; margin-bottom: 1rem; }
</style>
