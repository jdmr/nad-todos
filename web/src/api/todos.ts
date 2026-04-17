import client from './client'
import type { Todo } from '@/types/todo'

export async function fetchTodos(): Promise<Todo[]> {
  const res = await client.get<Todo[]>('/todos')
  return res.data
}

export async function createTodo(title: string): Promise<Todo> {
  const res = await client.post<Todo>('/todos', { title, completed: false })
  return res.data
}

export async function updateTodo(todo: Todo): Promise<Todo> {
  const res = await client.put<Todo>(`/todos/${todo.id}`, todo)
  return res.data
}

export async function deleteTodo(id: number): Promise<void> {
  await client.delete(`/todos/${id}`)
}
