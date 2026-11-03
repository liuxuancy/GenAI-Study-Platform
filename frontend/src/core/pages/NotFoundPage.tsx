// 404 页面：地址不存在时显示。
import { Button, Result } from 'antd'
import { Link } from 'react-router-dom'

import { useI18n } from '@/core/i18n/useI18n'

export function NotFoundPage() {
  const { t } = useI18n()

  return (
    <Result
      status="404"
      title={t('notFound.title')}
      extra={
        <Link to="/">
          <Button type="primary">{t('notFound.backHome')}</Button>
        </Link>
      }
    />
  )
}
