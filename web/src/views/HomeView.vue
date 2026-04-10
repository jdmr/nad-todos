<script setup lang="ts">
import { onMounted } from 'vue'
import { useTodoStore } from '@/stores/todos'
import TodoForm from '@/components/TodoForm.vue'
import TodoItem from '@/components/TodoItem.vue'

const store = useTodoStore()

onMounted(() => store.load())
</script>

<template>
  <main class="page">
    <header class="header">
      <h1 class="title">Todos</h1>
      <div class="rule" />
    </header>

    <TodoForm @add="store.add" />

    <Transition name="fade">
      <div v-if="store.error" class="error" role="alert">
        {{ store.error }}
      </div>
    </Transition>

    <div v-if="store.loading" class="loading" aria-label="Loading todos">
      <span class="dot" />
      <span class="dot" />
      <span class="dot" />
    </div>

    <template v-else>
      <TransitionGroup v-if="store.todos.length" name="list" tag="ul" class="list" appear>
        <TodoItem
          v-for="(todo, index) in store.todos"
          :key="todo.id"
          :todo="todo"
          :style="{ '--i': index }"
          @toggle="store.toggle"
          @remove="store.remove"
          @update="store.update"
        />
      </TransitionGroup>

      <div v-else class="empty">
        <p class="empty-title">Nothing here yet</p>
        <p class="empty-sub">Add your first task above</p>
      </div>

      <footer v-if="store.todos.length" class="footer">
        <span>{{ store.remaining }} remaining</span>
        <span class="footer-dot">&middot;</span>
        <span>{{ store.completedCount }} completed</span>
      </footer>
    </template>
  </main>
</template>

<style scoped>
.page {
  padding-top: 5rem;
  padding-bottom: 4rem;
}

.header {
  margin-bottom: 2.5rem;
}

.title {
  font-family: var(--font-display);
  font-size: clamp(2.5rem, 6vw, 3.5rem);
  font-weight: 400;
  letter-spacing: -0.02em;
  line-height: 1.1;
  color: var(--color-text);
}

.rule {
  width: 2.5rem;
  height: 3px;
  background: var(--color-accent);
  margin-top: 0.875rem;
  border-radius: 2px;
}

.error {
  margin-top: 1.5rem;
  padding: 0.75rem 1rem;
  background: var(--color-error-bg);
  color: var(--color-error-text);
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.loading {
  display: flex;
  justify-content: center;
  gap: 0.5rem;
  padding: 3rem 0;
}

.dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--color-text-muted);
  animation: pulse 1.2s ease-in-out infinite;
}

.dot:nth-child(2) {
  animation-delay: 0.15s;
}

.dot:nth-child(3) {
  animation-delay: 0.3s;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 0.3;
    transform: scale(0.8);
  }
  50% {
    opacity: 1;
    transform: scale(1);
  }
}

.list {
  list-style: none;
  padding: 0;
  margin-top: 2rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  position: relative;
}

.list-enter-active {
  animation: slide-in 0.35s cubic-bezier(0.16, 1, 0.3, 1) both;
  animation-delay: calc(var(--i, 0) * 0.04s);
}

.list-leave-active {
  animation: slide-out 0.25s ease-in forwards;
}

.list-move {
  transition: transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes slide-in {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes slide-out {
  from {
    opacity: 1;
    transform: translateX(0);
  }
  to {
    opacity: 0;
    transform: translateX(24px);
  }
}

.empty {
  text-align: center;
  padding: 4rem 0 2rem;
}

.empty-title {
  font-family: var(--font-display);
  font-size: 1.35rem;
  color: var(--color-text-muted);
}

.empty-sub {
  font-size: 0.875rem;
  color: var(--color-text-muted);
  margin-top: 0.35rem;
}

.footer {
  display: flex;
  justify-content: center;
  gap: 0.5rem;
  margin-top: 1.75rem;
  font-size: 0.75rem;
  color: var(--color-text-muted);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  font-weight: 500;
}

.footer-dot {
  opacity: 0.5;
}
</style>
