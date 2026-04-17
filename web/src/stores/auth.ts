import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as authApi from '@/api/auth'
import type { AuthUser, Role } from '@/api/auth'

export const useAuthStore = defineStore(
  'auth',
  () => {
    const user = ref<AuthUser | null>(null)
    const loading = ref(false)
    const error = ref<string | null>(null)

    const isAuthenticated = computed(() => user.value !== null)
    const isAdmin = computed(() => user.value?.role === 'admin')
    const isWebAuthnSupported = computed(() => authApi.isWebAuthnSupported())

    function hasRole(role: Role) {
      return user.value?.role === role
    }

    function setError(e: unknown, fallback: string) {
      if (e instanceof Error) {
        if (e.name === 'NotAllowedError') {
          error.value = 'Operation was cancelled'
        } else if (e.name === 'NotSupportedError') {
          error.value = 'WebAuthn is not supported in this browser'
        } else {
          error.value = e.message || fallback
        }
      } else {
        error.value = fallback
      }
    }

    async function login(email: string) {
      loading.value = true
      error.value = null
      try {
        user.value = await authApi.login(email)
      } catch (e) {
        setError(e, 'Login failed')
        throw e
      } finally {
        loading.value = false
      }
    }

    async function register(
      invitationToken: string,
      name: string,
      deviceName: string,
      email?: string,
    ) {
      loading.value = true
      error.value = null
      try {
        user.value = await authApi.register(invitationToken, name, deviceName, email)
      } catch (e) {
        setError(e, 'Registration failed')
        throw e
      } finally {
        loading.value = false
      }
    }

    async function logout() {
      try {
        await authApi.logout()
      } catch {
        // ignore
      } finally {
        user.value = null
      }
    }

    async function checkAuth(): Promise<boolean> {
      try {
        user.value = await authApi.getCurrentUser()
        return true
      } catch {
        user.value = null
        return false
      }
    }

    function clearUser() {
      user.value = null
    }

    return {
      user,
      loading,
      error,
      isAuthenticated,
      isAdmin,
      isWebAuthnSupported,
      hasRole,
      login,
      register,
      logout,
      checkAuth,
      clearUser,
    }
  },
  {
    persist: {
      key: 'todos-auth',
      pick: ['user'],
    },
  },
)
