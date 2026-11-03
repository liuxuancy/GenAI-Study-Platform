// 请求封装：自动带上 token，解开后端的 {code, msg, data}，失败时抛出 ApiError。
// 收到 401 时通知登录状态退出，页面随之跳回登录页。

export class ApiError extends Error {
  readonly code: number

  constructor(code: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code // 和后端 errs 的错误码一致：400、401、403、404、409、500；0 表示网络不通
  }
}

interface ApiResult<T> {
  code: number
  msg: string
  data: T
}

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export interface HttpAuthHooks {
  getToken: () => string | null
  onUnauthorized: () => void
}

let authHooks: HttpAuthHooks = {
  getToken: getNoToken,
  onUnauthorized: ignoreUnauthorized,
}

// setupHttpAuth 告诉请求封装去哪取 token、401 时做什么。这样 http 不依赖登录状态模块，避免循环引用。
// 全局只初始化一次：启动时由 main.tsx 调用。
export function setupHttpAuth(hooks: HttpAuthHooks) {
  authHooks = hooks
}

export async function request<T>(method: HttpMethod, path: string, body?: unknown): Promise<T> {
  const token = authHooks.getToken()
  const headers: Record<string, string> = {}
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }

  let response: Response
  try {
    response = await fetch(`/api${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(0, 'Network error: is the backend running?')
  }

  const result = await parseResult<T>(response)
  if (result.code !== 0) {
    // 带着 token 还被拒绝，说明登录已失效；登录接口本身的 401（密码错误）不算
    if (result.code === 401 && token) {
      authHooks.onUnauthorized()
    }
    throw new ApiError(result.code, result.msg)
  }
  return result.data
}

export const http = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
  patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, body),
  delete: <T>(path: string) => request<T>('DELETE', path),
}

export function uploadForm<T>(
  path: string,
  formData: FormData,
  onProgress?: (percent: number) => void,
): Promise<T> {
  return new Promise((resolve, reject) => {
    const token = authHooks.getToken()
    const xhr = new XMLHttpRequest()
    xhr.open('POST', `/api${path}`)
    if (token) {
      xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    }
    xhr.upload.onprogress = (event) => {
      if (!onProgress || !event.lengthComputable) {
        return
      }
      onProgress(Math.round((event.loaded / event.total) * 100))
    }
    xhr.onload = () => {
      try {
        const result = JSON.parse(xhr.responseText) as ApiResult<T>
        if (result.code !== 0) {
          if (result.code === 401 && token) {
            authHooks.onUnauthorized()
          }
          reject(new ApiError(result.code, result.msg))
          return
        }
        resolve(result.data)
      } catch {
        reject(new ApiError(xhr.status || 500, 'Unexpected server response'))
      }
    }
    xhr.onerror = () => reject(new ApiError(0, 'Network error: is the backend running?'))
    xhr.send(formData)
  })
}

// parseResult 响应体不是 JSON 时（比如后端没启动，代理返回 502）也转成统一格式。
async function parseResult<T>(response: Response): Promise<ApiResult<T>> {
  try {
    return (await response.json()) as ApiResult<T>
  } catch {
    return {
      code: response.status || 500,
      msg: `Unexpected server response (HTTP ${response.status})`,
      data: null as T,
    }
  }
}

function getNoToken() {
  return null
}

function ignoreUnauthorized() {}
