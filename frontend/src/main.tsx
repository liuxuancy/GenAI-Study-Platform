// 前端入口：先告诉请求封装怎么取 token、401 时怎么退出，再挂载页面。
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { App } from '@/App'
import { getAuthToken, handleUnauthorized } from '@/core/auth/store'
import { setupHttpAuth } from '@/core/http'
import '@/global.css'

setupHttpAuth({ getToken: getAuthToken, onUnauthorized: handleUnauthorized })

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
