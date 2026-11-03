import { ConfigProvider } from 'antd'
import enUS from 'antd/locale/en_US'
import zhCN from 'antd/locale/zh_CN'
import { RouterProvider } from 'react-router-dom'

import { appRouter } from '@/core/router'
import { useLocaleStore } from '@/core/i18n/store'
import { appTheme } from '@/core/theme'

export function App() {
  const locale = useLocaleStore((state) => state.locale)

  return (
    <ConfigProvider locale={locale === 'zh' ? zhCN : enUS} theme={appTheme}>
      <RouterProvider router={appRouter} />
    </ConfigProvider>
  )
}
