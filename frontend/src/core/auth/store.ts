// 登录状态：token 和当前用户。
// 勾选"记住我"时 token 存 localStorage，关掉浏览器仍保持登录；否则存 sessionStorage，关掉标签页就退出。
import { create } from 'zustand'

import * as accountApi from '@/core/api/account'
import type { CurrentUser, LoginInput } from '@/core/api/account'

const tokenStorageKey = 'caveclass.token'

interface AuthState {
  token: string | null
  currentUser: CurrentUser | null // 有 token 但还没请求 /me 时为 null
  login: (input: LoginInput, remember: boolean) => Promise<void>
  loadCurrentUser: () => Promise<void>
  logout: () => Promise<void>
  clearSession: () => void
}

export const useAuthStore = create<AuthState>()((set, get) => ({
  token: readStoredToken(),
  currentUser: null,

  async login(input, remember) {
    const result = await accountApi.login(input)
    saveToken(result.token, remember)
    set({ token: result.token, currentUser: null })
    await get().loadCurrentUser()
  },

  // loadCurrentUser 刷新页面后、登录成功后各调用一次，拿到角色和权限码
  async loadCurrentUser() {
    const currentUser = await accountApi.getCurrentUser()
    set({ currentUser })
  },

  async logout() {
    try {
      await accountApi.logout()
    } catch {
      // 后端只记日志，请求失败也照样退出
    } finally {
      get().clearSession()
    }
  },

  // clearSession 只清本地状态，不调接口。token 失效（401）时也走这里。
  clearSession() {
    removeStoredToken()
    set({ token: null, currentUser: null })
  },
}))

// getAuthToken 给请求封装用，在组件外读取当前 token。
export function getAuthToken() {
  return useAuthStore.getState().token
}

// handleUnauthorized 给请求封装用：收到 401 时退出登录。
export function handleUnauthorized() {
  useAuthStore.getState().clearSession()
}

function readStoredToken() {
  return localStorage.getItem(tokenStorageKey) ?? sessionStorage.getItem(tokenStorageKey)
}

function saveToken(token: string, remember: boolean) {
  removeStoredToken()
  const storage = remember ? localStorage : sessionStorage
  storage.setItem(tokenStorageKey, token)
}

function removeStoredToken() {
  localStorage.removeItem(tokenStorageKey)
  sessionStorage.removeItem(tokenStorageKey)
}
