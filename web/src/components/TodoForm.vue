<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{ add: [title: string] }>()

const title = ref('')

function submit() {
  const trimmed = title.value.trim()
  if (!trimmed) return
  emit('add', trimmed)
  title.value = ''
}
</script>

<template>
  <form @submit.prevent="submit" class="form">
    <div class="input-wrap">
      <input
        v-model="title"
        type="text"
        placeholder="What needs to be done?"
        aria-label="New todo title"
        class="input"
      />
    </div>
    <button type="submit" class="btn" :disabled="!title.trim()">Add</button>
  </form>
</template>

<style scoped>
.form {
  display: flex;
  gap: 1rem;
  align-items: flex-end;
}

.input-wrap {
  flex: 1;
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
  font-size: 1rem;
  color: var(--color-text);
  background: transparent;
  border: none;
  border-bottom: 2px solid var(--color-border);
  outline: none;
  transition: border-color 0.2s ease;
}

.input::placeholder {
  color: var(--color-text-muted);
}

.input:focus {
  border-bottom-color: transparent;
}

.btn {
  padding: 0.5rem 1.5rem;
  font-family: var(--font-body);
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: #fff;
  background: var(--color-accent);
  border: none;
  border-radius: 100px;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.btn:hover:not(:disabled) {
  background: var(--color-accent-hover);
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

.btn:active:not(:disabled) {
  transform: translateY(0) scale(0.97);
  box-shadow: none;
}

.btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}
</style>
