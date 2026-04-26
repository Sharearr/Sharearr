import { defineStore } from 'pinia'

export interface AuthUser {
  id: number
  username: string
  roles: string[]
  features: string[]
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    // authn
    user: null as AuthUser | null,
    expiresAt: null as string | null,
    // authz
  }),
  getters: {
    isAuthenticated: (state) => state.user !== null && state.expiresAt !== null,
    isAdmin: (state) => state.user?.roles.includes('admin') ?? false,
  },
  actions: {
    setAuth(user: AuthUser, expiresAt: string) {
      this.user = user
      this.expiresAt = expiresAt
    },
    clearAuth() {
      this.user = null
      this.expiresAt = null
    },
  },
})
