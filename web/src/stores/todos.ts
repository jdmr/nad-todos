import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import type { Todo } from '@/types/todo'
import * as api from '@/api/todos'

export const useTodoStore = defineStore('todos', () => {
  const todos = ref<Todo[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const remaining = computed(() => todos.value.filter((t) => !t.completed).length)
  const completedCount = computed(() => todos.value.filter((t) => t.completed).length)

  async function load() {
    loading.value = true
    error.value = null
    try {
      todos.value = await api.fetchTodos()
    } catch (e) {
      error.value = (e as Error).message
    } finally {
      loading.value = false
    }
  }

  async function add(title: string) {
    error.value = null
    try {
      const todo = await api.createTodo(title)
      todos.value.push(todo)
    } catch (e) {
      error.value = (e as Error).message
    }
  }

  async function toggle(todo: Todo) {
    error.value = null
    try {
      const updated = await api.updateTodo({ ...todo, completed: !todo.completed })
      const idx = todos.value.findIndex((t) => t.id === todo.id)
      if (idx !== -1) todos.value[idx] = updated
    } catch (e) {
      error.value = (e as Error).message
    }
  }

  async function remove(id: number) {
    error.value = null
    try {
      await api.deleteTodo(id)
      todos.value = todos.value.filter((t) => t.id !== id)
    } catch (e) {
      error.value = (e as Error).message
    }
  }

  async function update(todo: Todo) {
    error.value = null
    try {
      const updated = await api.updateTodo(todo)
      const idx = todos.value.findIndex((t) => t.id === todo.id)
      if (idx !== -1) todos.value[idx] = updated
    } catch (e) {
      error.value = (e as Error).message
    }
  }

  return { todos, loading, error, remaining, completedCount, load, add, toggle, remove, update }
})
