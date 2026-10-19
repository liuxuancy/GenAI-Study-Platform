// 根组件：套上 Ant Design 主题，挂载路由。
import { ConfigProvider } from 'antd'
import { RouterProvider } from 'react-router-dom'

import { appRouter } from '@/core/router'
import { appTheme } from '@/core/theme'

export function App() {
  return (
    <ConfigProvider theme={appTheme}>
      <RouterProvider router={appRouter} />
    </ConfigProvider>
  )
}
