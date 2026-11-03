// 登录后的页面框架：顶栏显示当前用户和退出按钮，左侧菜单，右侧是子路由页面。
import { Button, Layout, Menu, Tag } from 'antd'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import { LanguageSwitcher } from '@/core/i18n/LanguageSwitcher'
import { localizedRoleName } from '@/core/i18n/roleName'
import { useI18n } from '@/core/i18n/useI18n'
import styles from '@/core/layout/AppLayout.module.css'

function menuFor(role: string | undefined, t: (key: string) => string) {
  if (role === 'ROLE_ADMIN') {
    return [
      { key: '/admin', label: t('layout.admin.users') },
      { key: '/admin/classes', label: t('layout.admin.classes') },
      { key: '/admin/courses', label: t('layout.admin.courses') },
      { key: '/admin/system', label: t('layout.admin.system') },
      { key: '/account', label: t('layout.admin.account') },
    ]
  }
  if (role === 'ROLE_TEACHER') {
    return [
      { key: '/teacher', label: t('layout.teacher.courses') },
      { key: '/teacher/account', label: t('layout.teacher.account') },
    ]
  }
  if (role === 'ROLE_STUDENT') {
    return [
      { key: '/student', label: t('layout.student.courses') },
      { key: '/student/account', label: t('layout.student.account') },
    ]
  }
  return [{ key: '/', label: t('layout.home') }]
}

export function AppLayout() {
  const currentUser = useAuthStore((state) => state.currentUser)
  const logout = useAuthStore((state) => state.logout)
  const navigate = useNavigate()
  const location = useLocation()
  const { t } = useI18n()
  const menuItems = menuFor(currentUser?.role, t)

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Layout.Header className={styles.header}>
        <div className={styles.brand}>
          <span className={styles.mark} aria-hidden="true">
            C
          </span>
          <span className={styles.brandName}>{t('common.brandName')}</span>
        </div>
        <div className={styles.user}>
          <span>{currentUser?.nickname || currentUser?.username}</span>
          {currentUser ? <Tag>{localizedRoleName(currentUser.role)}</Tag> : null}
          <LanguageSwitcher compact />
          <Button onClick={logout}>{t('common.signOut')}</Button>
        </div>
      </Layout.Header>
      <Layout>
        <Layout.Sider width={220} className={styles.sider}>
          <Menu
            mode="inline"
            items={menuItems}
            selectedKeys={[location.pathname]}
            onClick={(item) => navigate(item.key)}
            style={{ borderInlineEnd: 0, paddingTop: 12 }}
          />
        </Layout.Sider>
        <Layout.Content className={styles.content}>
          <Outlet />
        </Layout.Content>
      </Layout>
    </Layout>
  )
}
