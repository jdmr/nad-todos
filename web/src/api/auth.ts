import {
  startRegistration,
  startAuthentication,
  browserSupportsWebAuthn,
} from '@simplewebauthn/browser'
import type {
  PublicKeyCredentialCreationOptionsJSON,
  PublicKeyCredentialRequestOptionsJSON,
} from '@simplewebauthn/browser'
import client from './client'

export type Role = 'user' | 'admin'

export interface AuthUser {
  user_id: string
  email: string
  name: string
  role: Role
}

export interface InvitationInfo {
  email: string
  default_role: Role
  is_bootstrap: boolean
}

export interface SessionItem {
  id: string
  device_name: string
  ip_address: string
  created_at: string
  last_activity_at: string
  is_current: boolean
}

export interface CredentialItem {
  id: string
  device_name: string
  created_at: string
  last_used_at?: string | null
}

export function isWebAuthnSupported(): boolean {
  return browserSupportsWebAuthn()
}

export function getDefaultDeviceName(): string {
  const ua = navigator.userAgent
  if (ua.includes('iPhone')) return 'iPhone'
  if (ua.includes('iPad')) return 'iPad'
  if (ua.includes('Android')) return 'Android Device'
  if (ua.includes('Mac')) return 'Mac'
  if (ua.includes('Windows')) return 'Windows PC'
  if (ua.includes('Linux')) return 'Linux'
  return 'Unknown Device'
}

export async function getInvitation(token: string): Promise<InvitationInfo> {
  const res = await client.get<InvitationInfo>(
    `/auth/invitations/${encodeURIComponent(token)}`,
  )
  return res.data
}

interface ChallengeResp<T> {
  challenge_id: string
  challenge: { publicKey: T } | T
}

function unwrapOptions<T>(payload: ChallengeResp<T>['challenge']): T {
  return payload && typeof payload === 'object' && 'publicKey' in payload
    ? (payload as { publicKey: T }).publicKey
    : (payload as T)
}

export async function register(
  invitationToken: string,
  name: string,
  deviceName: string,
  email?: string,
): Promise<AuthUser> {
  const start = await client.post<ChallengeResp<PublicKeyCredentialCreationOptionsJSON>>(
    '/auth/register/options',
    { invitation_token: invitationToken, name, device_name: deviceName, email },
  )
  const optionsJSON = unwrapOptions(start.data.challenge)
  const credential = await startRegistration({ optionsJSON })
  const verify = await client.post<AuthUser>('/auth/register/verify', {
    challenge_id: start.data.challenge_id,
    credential,
    device_name: deviceName,
  })
  return verify.data
}

export async function login(email: string): Promise<AuthUser> {
  const start = await client.post<ChallengeResp<PublicKeyCredentialRequestOptionsJSON>>(
    '/auth/login/options',
    { email },
  )
  const optionsJSON = unwrapOptions(start.data.challenge)
  const credential = await startAuthentication({ optionsJSON })
  const verify = await client.post<AuthUser>('/auth/login/verify', {
    challenge_id: start.data.challenge_id,
    credential,
  })
  return verify.data
}

export async function getCurrentUser(): Promise<AuthUser> {
  const res = await client.get<AuthUser>('/auth/me')
  return res.data
}

export async function logout(): Promise<void> {
  await client.post('/auth/logout')
}

export async function listSessions(): Promise<{ sessions: SessionItem[] }> {
  const res = await client.get<{ sessions: SessionItem[] }>('/auth/sessions')
  return res.data
}

export async function revokeSession(sessionId: string): Promise<void> {
  await client.post('/auth/sessions/revoke', { session_id: sessionId })
}

export async function listCredentials(): Promise<{ credentials: CredentialItem[] }> {
  const res = await client.get<{ credentials: CredentialItem[] }>('/auth/credentials')
  return res.data
}

export async function addCredential(deviceName: string): Promise<CredentialItem> {
  const start = await client.post<ChallengeResp<PublicKeyCredentialCreationOptionsJSON>>(
    '/auth/credentials/options',
    { device_name: deviceName },
  )
  const optionsJSON = unwrapOptions(start.data.challenge)
  const credential = await startRegistration({ optionsJSON })
  const verify = await client.post<CredentialItem>('/auth/credentials/verify', {
    challenge_id: start.data.challenge_id,
    credential,
    device_name: deviceName,
  })
  return verify.data
}

export async function deleteCredential(id: string): Promise<void> {
  await client.delete(`/auth/credentials/${encodeURIComponent(id)}`)
}
