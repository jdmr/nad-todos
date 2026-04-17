<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const email = ref('')

async function submit() {
  if (!email.value) return
  try {
    await auth.login(email.value)
    router.push('/')
  } catch {
    // error already shown
  }
}
</script>

<template>
  <main class="page">
    <header class="header">
      <p class="eyebrow">Welcome back</p>
      <h1 class="title">Sign in</h1>
      <div class="rule" />
    </header>

    <form class="form" @submit.prevent="submit">
      <div class="field">
        <label class="label" for="email">Email</label>
        <div class="input-wrap">
          <input
            id="email"
            v-model="email"
            type="email"
            autocomplete="username webauthn"
            required
            class="input"
            placeholder="you@example.com"
          />
        </div>
      </div>

      <button type="submit" class="button" :disabled="auth.loading">
        <span class="button-label">
          {{ auth.loading ? 'Signing in…' : 'Sign in with passkey' }}
        </span>
      </button>

      <Transition name="fade">
        <p v-if="auth.error" class="error" role="alert">{{ auth.error }}</p>
      </Transition>

      <p v-if="!auth.isWebAuthnSupported" class="warn">
        Your browser doesn't support WebAuthn. You'll need a modern browser to sign in.
      </p>
    </form>
  </main>
</template>

<style scoped>
.page {
  padding-top: 5rem;
  padding-bottom: 4rem;
  max-width: 26rem;
}

.header {
  margin-bottom: 2.75rem;
}

.eyebrow {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.18em;
  color: var(--color-text-muted);
  margin-bottom: 0.6rem;
}

.title {
  font-family: var(--font-display);
  font-size: clamp(2.5rem, 6vw, 3.25rem);
  font-weight: 400;
  letter-spacing: -0.02em;
  line-height: 1.05;
  color: var(--color-text);
}

.rule {
  width: 2.5rem;
  height: 3px;
  background: var(--color-accent);
  margin-top: 0.875rem;
  border-radius: 2px;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 1.75rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.label {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--color-text-muted);
  font-weight: 500;
}

.input-wrap {
  position: relative;
}

.input-wrap::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 50%;
  width: 0;
  height: 2px;
  background: var(--color-accent);
  transition: all 0.35s cubic-bezier(0.16, 1, 0.3, 1);
}

.input-wrap:focus-within::after {
  left: 0;
  width: 100%;
}

.input {
  width: 100%;
  padding: 0.625rem 0;
  font-family: var(--font-body);
  font-size: 1.0625rem;
  color: var(--color-text);
  background: transparent;
  border: none;
  border-bottom: 2px solid var(--color-border);
  outline: none;
  transition: border-color 0.2s ease;
}

.input::placeholder {
  color: var(--color-text-muted);
  font-style: italic;
}

.input:hover {
  border-bottom-color: var(--color-border-hover);
}

.input:focus {
  border-bottom-color: transparent;
}

.input:-webkit-autofill,
.input:-webkit-autofill:hover,
.input:-webkit-autofill:focus {
  -webkit-text-fill-color: var(--color-text);
  -webkit-box-shadow: 0 0 0 1000px var(--color-bg) inset;
  caret-color: var(--color-text);
  transition: background-color 9999s ease-out;
}

.button {
  position: relative;
  margin-top: 0.5rem;
  padding: 0.95rem 1.25rem;
  background: var(--color-text);
  color: var(--color-bg);
  border: none;
  border-radius: 999px;
  font-family: var(--font-body);
  font-size: 0.78rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  cursor: pointer;
  transition:
    transform 0.18s cubic-bezier(0.16, 1, 0.3, 1),
    box-shadow 0.18s ease,
    opacity 0.18s ease;
  box-shadow: var(--shadow-sm);
}

.button:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

.button:active:not(:disabled) {
  transform: translateY(0) scale(0.99);
  box-shadow: var(--shadow-sm);
}

.button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.button-label {
  display: inline-block;
}

.error {
  color: var(--color-error-text);
  background: var(--color-error-bg);
  padding: 0.65rem 0.85rem;
  border-radius: 6px;
  font-size: 0.875rem;
  border-left: 2px solid var(--color-accent);
}

.warn {
  font-size: 0.8rem;
  color: var(--color-text-muted);
  font-style: italic;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
