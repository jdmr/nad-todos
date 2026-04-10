<script setup lang="ts">
import { ref, nextTick } from 'vue'
import type { Todo } from '@/types/todo'

const props = defineProps<{ todo: Todo }>()
const emit = defineEmits<{
  toggle: [todo: Todo]
  remove: [id: number]
  update: [todo: Todo]
}>()

const editing = ref(false)
const editTitle = ref('')
const editInput = ref<HTMLInputElement | null>(null)

function startEdit() {
  editing.value = true
  editTitle.value = props.todo.title
  nextTick(() => editInput.value?.focus())
}

function saveEdit() {
  const trimmed = editTitle.value.trim()
  if (trimmed && trimmed !== props.todo.title) {
    emit('update', { ...props.todo, title: trimmed })
  }
  editing.value = false
}

function cancelEdit() {
  editing.value = false
}
</script>

<template>
  <li class="item" :class="{ 'item--done': todo.completed }">
    <label class="check">
      <input
        type="checkbox"
        :checked="todo.completed"
        @change="emit('toggle', todo)"
        :aria-label="`Mark &quot;${todo.title}&quot; as ${todo.completed ? 'incomplete' : 'complete'}`"
      />
      <span class="check-box">
        <svg viewBox="0 0 12 10" fill="none">
          <polyline points="1.5 5.5 4.5 8.5 10.5 1.5" />
        </svg>
      </span>
    </label>

    <template v-if="editing">
      <input
        ref="editInput"
        v-model="editTitle"
        @keyup.enter="saveEdit"
        @keyup.escape="cancelEdit"
        @blur="saveEdit"
        aria-label="Edit todo title"
        class="edit-input"
      />
    </template>
    <template v-else>
      <span @dblclick="startEdit" class="title">
        {{ todo.title }}
      </span>
    </template>

    <div class="actions">
      <button
        v-if="!editing"
        @click="startEdit"
        :aria-label="`Edit &quot;${todo.title}&quot;`"
        class="action action--edit"
      >
        <svg viewBox="0 0 16 16" fill="none" width="14" height="14">
          <path
            d="M11.5 1.5l3 3-9 9H2.5v-3l9-9z"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
      <button
        @click="emit('remove', todo.id)"
        :aria-label="`Delete &quot;${todo.title}&quot;`"
        class="action action--delete"
      >
        <svg viewBox="0 0 16 16" fill="none" width="14" height="14">
          <path
            d="M4 4l8 8M12 4l-8 8"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
          />
        </svg>
      </button>
    </div>
  </li>
</template>

<style scoped>
.item {
  display: flex;
  align-items: center;
  gap: 0.875rem;
  padding: 0.8rem 1rem;
  background: var(--color-surface);
  border-radius: 10px;
  box-shadow: var(--shadow-sm);
  transition:
    box-shadow 0.2s ease,
    background 0.2s ease;
}

.item:hover {
  box-shadow: var(--shadow-md);
}

/* ── Custom checkbox ── */
.check {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  flex-shrink: 0;
}

.check input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
}

.check-box {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid var(--color-border-hover);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.25s cubic-bezier(0.16, 1, 0.3, 1);
  background: transparent;
}

.check-box svg {
  width: 12px;
  height: 10px;
}

.check-box svg polyline {
  stroke: #fff;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-dasharray: 20;
  stroke-dashoffset: 20;
  transition: stroke-dashoffset 0.3s cubic-bezier(0.16, 1, 0.3, 1) 0.05s;
}

.check:hover .check-box {
  border-color: var(--color-check);
}

.check input:focus-visible + .check-box {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

.item--done .check-box {
  background: var(--color-check);
  border-color: var(--color-check);
  animation: check-pop 0.35s cubic-bezier(0.16, 1, 0.3, 1);
}

.item--done .check-box svg polyline {
  stroke-dashoffset: 0;
}

@keyframes check-pop {
  0% {
    transform: scale(1);
  }
  40% {
    transform: scale(1.2);
  }
  100% {
    transform: scale(1);
  }
}

/* ── Title ── */
.title {
  flex: 1;
  font-size: 0.95rem;
  color: var(--color-text);
  user-select: none;
  position: relative;
  transition: color 0.25s ease;
  cursor: default;
}

.item--done .title {
  color: var(--color-text-muted);
}

.item--done .title::after {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  height: 1.5px;
  background: var(--color-text-muted);
  animation: strike 0.3s ease-out forwards;
}

@keyframes strike {
  from {
    width: 0;
  }
  to {
    width: 100%;
  }
}

/* ── Edit input ── */
.edit-input {
  flex: 1;
  padding: 0.2rem 0;
  font-family: var(--font-body);
  font-size: 0.95rem;
  color: var(--color-text);
  background: transparent;
  border: none;
  border-bottom: 2px solid var(--color-accent);
  outline: none;
  border-radius: 0;
}

/* ── Action buttons ── */
.actions {
  display: flex;
  gap: 0.125rem;
  margin-left: auto;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.item:hover .actions,
.item:focus-within .actions {
  opacity: 1;
}

.action {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  color: var(--color-text-muted);
  transition: all 0.15s ease;
}

.action--edit:hover {
  background: var(--color-border);
  color: var(--color-accent);
}

.action--delete:hover {
  background: var(--color-border);
  color: var(--color-danger);
}
</style>
