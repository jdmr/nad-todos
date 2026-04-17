<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  listSessions,
  revokeSession,
  listCredentials,
  addCredential,
  deleteCredential,
  getDefaultDeviceName,
} from '@/api/auth'
import type { SessionItem, CredentialItem } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const sessions = ref<SessionItem[]>([])
const credentials = ref<CredentialItem[]>([])
const newDeviceName = ref(getDefaultDeviceName())
const error = ref<string | null>(null)
const busy = ref(false)

async function refresh() {
  try {
    sessions.value = (await listSessions()).sessions
    credentials.value = (await listCredentials()).credentials
  } catch {
    error.value = 'Failed to load account info'
  }
}

onMounted(refresh)

async function revoke(id: string) {
  await revokeSession(id)
  await refresh()
}

async function addPasskey() {
  if (!newDeviceName.value) return
  busy.value = true
  error.value = null
  try {
    await addCredential(newDeviceName.value)
    await refresh()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to add passkey'
  } finally {
    busy.value = false
  }
}

async function removePasskey(id: string) {
  if (credentials.value.length <= 1) {
    error.value = 'Cannot delete the last passkey'
    return
  }
  await deleteCredential(id)
  await refresh()
}
</script>

<template>
  <main class="page">
    <h1 class="title">Account</h1>
    <div class="rule" />

    <p v-if="auth.user" class="who">
      {{ auth.user.email }} — <em>{{ auth.user.role }}</em>
    </p>

    <p v-if="error" class="error">{{ error }}</p>

    <section class="section">
      <h2>Passkeys</h2>
      <ul v-if="credentials.length" class="list">
        <li v-for="c in credentials" :key="c.id" class="row">
          <span>{{ c.device_name || 'Passkey' }}</span>
          <button class="link" @click="removePasskey(c.id)">remove</button>
        </li>
      </ul>
      <p v-else class="muted">No passkeys yet.</p>

      <div class="add">
        <input v-model="newDeviceName" class="input" placeholder="Device name" />
        <button class="button" :disabled="busy" @click="addPasskey">
          {{ busy ? 'Adding…' : 'Add passkey' }}
        </button>
      </div>
    </section>

    <section class="section">
      <h2>Active sessions</h2>
      <ul v-if="sessions.length" class="list">
        <li v-for="s in sessions" :key="s.id" class="row">
          <span>
            {{ s.device_name || 'Unknown device' }}
            <span class="muted">— last active {{ new Date(s.last_activity_at).toLocaleString() }}</span>
            <span v-if="s.is_current" class="badge">this session</span>
          </span>
          <button v-if="!s.is_current" class="link" @click="revoke(s.id)">revoke</button>
        </li>
      </ul>
    </section>
  </main>
</template>

<style scoped>
.page { padding-top: 4rem; max-width: 36rem; }
.title { font-family: var(--font-display); font-size: clamp(2rem, 5vw, 2.75rem); font-weight: 400; letter-spacing: -0.02em; }
.rule { width: 2.5rem; height: 3px; background: var(--color-accent); margin: 0.875rem 0 2rem; border-radius: 2px; }
.who { color: var(--color-text-muted); margin-bottom: 1.5rem; }
.section { margin-top: 2rem; }
.section h2 { font-family: var(--font-display); font-weight: 400; font-size: 1.25rem; margin-bottom: 0.75rem; }
.list { list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 0.5rem; }
.row { display: flex; justify-content: space-between; align-items: center; padding: 0.5rem 0.75rem; border: 1px solid var(--color-border, #eee); border-radius: 6px; }
.muted { color: var(--color-text-muted); font-size: 0.85rem; }
.badge { margin-left: 0.5rem; font-size: 0.7rem; padding: 0.125rem 0.4rem; background: var(--color-accent); color: var(--color-bg); border-radius: 3px; }
.link { background: none; border: none; color: var(--color-error-text, #c00); cursor: pointer; font-size: 0.85rem; }
.add { display: flex; gap: 0.5rem; margin-top: 0.75rem; }
.input { flex: 1; padding: 0.5rem 0.75rem; border: 1px solid var(--color-border, #ddd); border-radius: 6px; }
.button { padding: 0.5rem 0.875rem; background: var(--color-text); color: var(--color-bg); border: none; border-radius: 6px; cursor: pointer; }
.button:disabled { opacity: 0.6; }
.error { color: var(--color-error-text, #c00); background: var(--color-error-bg, #fee); padding: 0.5rem 0.75rem; border-radius: 6px; }
</style>
