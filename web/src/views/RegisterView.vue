<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { getInvitation, getDefaultDeviceName } from '@/api/auth'
import type { InvitationInfo } from '@/api/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const token = computed(() => String(route.query.token ?? ''))
const invitation = ref<InvitationInfo | null>(null)
const lookupError = ref<string | null>(null)
const name = ref('')
const email = ref('')
const deviceName = ref(getDefaultDeviceName())

onMounted(async () => {
  if (!token.value) {
    lookupError.value = 'Missing invitation token'
    return
  }
  try {
    invitation.value = await getInvitation(token.value)
  } catch {
    lookupError.value = 'Invalid or expired invitation'
  }
})

watch(invitation, (inv) => {
  if (inv && !inv.is_bootstrap) {
    email.value = inv.email
  }
})

const emailEditable = computed(() => invitation.value?.is_bootstrap ?? false)
const isBootstrap = computed(() => invitation.value?.is_bootstrap ?? false)

const eyebrow = computed(() =>
  isBootstrap.value ? 'Founding entry · No. 01' : 'You have been invited',
)
const titleText = computed(() => (isBootstrap.value ? 'Inaugural admin' : 'Welcome aboard'))
const subtitleText = computed(() =>
  isBootstrap.value
    ? 'Your name will mark the opening of this archive.'
    : 'Add your details and bind a passkey to this device.',
)

async function submit() {
  if (!name.value || !deviceName.value) return
  if (emailEditable.value && !email.value) return
  try {
    await auth.register(
      token.value,
      name.value,
      deviceName.value,
      emailEditable.value ? email.value : undefined,
    )
    router.push('/')
  } catch {
    // error already shown
  }
}
</script>

<template>
  <main class="page">
    <span v-if="invitation" class="marginalia" aria-hidden="true">
      {{ isBootstrap ? '№ 1' : '№' }}
    </span>

    <header class="header" :class="{ ready: !!invitation }">
      <p class="eyebrow">{{ eyebrow }}</p>
      <h1 class="title" :class="{ bootstrap: isBootstrap }">{{ titleText }}</h1>
      <div class="rule" />
      <p v-if="invitation" class="subtitle">{{ subtitleText }}</p>
    </header>

    <p v-if="lookupError" class="error" role="alert">{{ lookupError }}</p>

    <form v-if="invitation" class="form" @submit.prevent="submit">
      <!-- Field I: Email -->
      <section class="field" :style="{ '--i': 0 }">
        <div class="field-head">
          <span class="numeral">I</span>
          <label class="label" for="email">Email</label>
        </div>

        <div v-if="!emailEditable" class="inscription">
          <span class="quote-mark" aria-hidden="true">“</span>
          <span class="inscription-text">{{ email }}</span>
          <span class="quote-mark close" aria-hidden="true">”</span>
        </div>
        <input
          v-if="!emailEditable"
          id="email"
          :value="email"
          type="hidden"
        />
        <div v-if="!emailEditable" class="attribution">
          inscribed by the inviting admin
        </div>

        <div v-else class="input-wrap">
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
      </section>

      <!-- Field II: Name -->
      <section class="field" :style="{ '--i': 1 }">
        <div class="field-head">
          <span class="numeral">II</span>
          <label class="label" for="name">Your name</label>
        </div>
        <div class="input-wrap">
          <input id="name" v-model="name" required class="input" placeholder="Ada Lovelace" />
        </div>
      </section>

      <!-- Field III: Device -->
      <section class="field" :style="{ '--i': 2 }">
        <div class="field-head">
          <span class="numeral">III</span>
          <label class="label" for="device">Device</label>
        </div>
        <div class="input-wrap">
          <input id="device" v-model="deviceName" required class="input" />
        </div>
        <p class="field-note">
          A label for the passkey saved on this machine.
        </p>
      </section>

      <div class="role-row" :style="{ '--i': 3 }">
        <span class="role-dot" aria-hidden="true" />
        <span class="role-label">Joining as</span>
        <span class="role-value">{{ invitation.default_role }}</span>
      </div>

      <button
        type="submit"
        class="button"
        :disabled="auth.loading"
        :style="{ '--i': 4 }"
      >
        <span class="button-label">
          {{ auth.loading ? 'Sealing…' : isBootstrap ? 'Inscribe & enter' : 'Create passkey' }}
        </span>
        <span class="button-arrow" aria-hidden="true">→</span>
      </button>

      <Transition name="fade">
        <p v-if="auth.error" class="error" role="alert">{{ auth.error }}</p>
      </Transition>
    </form>
  </main>
</template>

<style scoped>
.page {
  position: relative;
  padding-top: 5rem;
  padding-bottom: 4rem;
  max-width: 28rem;
}

/* Marginal mark — a small archival flourish in the negative space */
.marginalia {
  position: absolute;
  top: 5.6rem;
  left: -3rem;
  font-family: var(--font-display);
  font-size: 0.85rem;
  font-style: italic;
  color: var(--color-text-muted);
  letter-spacing: 0.04em;
  white-space: nowrap;
  opacity: 0;
  animation: settle 0.7s 0.05s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}

@media (max-width: 540px) {
  .marginalia {
    left: 0;
    top: 3.6rem;
    font-size: 0.75rem;
  }
}

.header {
  margin-bottom: 2.5rem;
  opacity: 0;
}
.header.ready {
  animation: settle 0.55s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}

.eyebrow {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.18em;
  color: var(--color-text-muted);
  margin-bottom: 0.65rem;
  font-weight: 500;
}

.title {
  font-family: var(--font-display);
  font-size: clamp(2.5rem, 6vw, 3.25rem);
  font-weight: 400;
  letter-spacing: -0.02em;
  line-height: 1.05;
  color: var(--color-text);
}

.title.bootstrap {
  font-style: italic;
}

.rule {
  width: 2.5rem;
  height: 3px;
  background: var(--color-accent);
  margin-top: 0.875rem;
  border-radius: 2px;
}

.subtitle {
  margin-top: 1.1rem;
  font-family: var(--font-display);
  font-style: italic;
  font-size: 1.05rem;
  line-height: 1.5;
  color: var(--color-text-secondary);
  max-width: 24rem;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 1.65rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  opacity: 0;
  animation: settle 0.55s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  animation-delay: calc(0.1s + var(--i, 0) * 0.07s);
}

.field-head {
  display: flex;
  align-items: baseline;
  gap: 0.65rem;
}

.numeral {
  font-family: var(--font-display);
  font-style: italic;
  font-size: 0.85rem;
  color: var(--color-accent);
  letter-spacing: 0.02em;
  min-width: 1.5rem;
}

.label {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.14em;
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

/* Inscription — locked email rendered as an editorial quotation */
.inscription {
  position: relative;
  padding: 0.5rem 0 0.4rem;
  font-family: var(--font-display);
  font-style: italic;
  font-size: 1.35rem;
  line-height: 1.3;
  color: var(--color-text);
  word-break: break-word;
}

.quote-mark {
  font-family: var(--font-display);
  font-style: italic;
  color: var(--color-accent);
  font-size: 1.65rem;
  line-height: 0;
  position: relative;
  top: 0.18em;
  margin-right: 0.05em;
}

.quote-mark.close {
  margin-left: 0.05em;
  margin-right: 0;
}

.inscription-text {
  font-feature-settings: 'liga' on, 'calt' on;
}

.attribution {
  font-size: 0.72rem;
  color: var(--color-text-muted);
  font-style: italic;
  letter-spacing: 0.02em;
  padding-top: 0.25rem;
  border-top: 1px dashed var(--color-border);
  margin-top: 0.35rem;
  width: fit-content;
  padding-right: 0.75rem;
}

.field-note {
  font-size: 0.72rem;
  color: var(--color-text-muted);
  font-style: italic;
  margin-top: 0.1rem;
}

/* Role row — a small typographic badge */
.role-row {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  padding: 0.85rem 0;
  border-top: 1px solid var(--color-border);
  border-bottom: 1px solid var(--color-border);
  margin-top: 0.25rem;
  opacity: 0;
  animation: settle 0.55s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  animation-delay: calc(0.1s + var(--i, 0) * 0.07s);
}

.role-dot {
  width: 7px;
  height: 7px;
  background: var(--color-accent);
  border-radius: 50%;
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--color-accent) 12%, transparent);
}

.role-label {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.14em;
  color: var(--color-text-muted);
  font-weight: 500;
}

.role-value {
  font-family: var(--font-display);
  font-style: italic;
  font-size: 1.05rem;
  color: var(--color-text);
  margin-left: auto;
}

.button {
  position: relative;
  margin-top: 0.5rem;
  padding: 0.95rem 1.4rem;
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
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.6rem;
  opacity: 0;
  animation: settle 0.55s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  animation-delay: calc(0.1s + var(--i, 0) * 0.07s);
}

.button:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

.button:hover:not(:disabled) .button-arrow {
  transform: translateX(3px);
}

.button:active:not(:disabled) {
  transform: translateY(0) scale(0.99);
  box-shadow: var(--shadow-sm);
}

.button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.button-arrow {
  display: inline-block;
  transition: transform 0.25s cubic-bezier(0.16, 1, 0.3, 1);
  font-size: 0.95rem;
  letter-spacing: 0;
}

.error {
  color: var(--color-error-text);
  background: var(--color-error-bg);
  padding: 0.65rem 0.85rem;
  border-radius: 6px;
  font-size: 0.875rem;
  border-left: 2px solid var(--color-accent);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@keyframes settle {
  0% {
    opacity: 0;
    transform: translateY(6px);
  }
  100% {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .marginalia,
  .header,
  .field,
  .role-row,
  .button {
    animation: none !important;
    opacity: 1 !important;
    transform: none !important;
  }
}
</style>
