import client from './client'
import type { Role } from './auth'

export interface AdminUser {
  id: string
  email: string
  name: string
  role: Role
  status: string
  created_at: string
}

export interface AdminInvitation {
  id: string
  email: string
  token: string
  default_role: Role
  status: string
  created_at: string
  expires_at: string
}

export async function listUsers(): Promise<AdminUser[]> {
  const res = await client.get<AdminUser[]>('/admin/users')
  return res.data
}

export async function setUserRole(userId: string, role: Role): Promise<void> {
  await client.put(`/admin/users/${userId}/role`, { role })
}

export async function listInvitations(): Promise<AdminInvitation[]> {
  const res = await client.get<AdminInvitation[]>('/admin/invitations')
  return res.data
}

export async function createInvitation(email: string, role: Role): Promise<AdminInvitation> {
  const res = await client.post<AdminInvitation>('/admin/invitations', { email, role })
  return res.data
}

export async function revokeInvitation(id: string): Promise<void> {
  await client.delete(`/admin/invitations/${id}`)
}
