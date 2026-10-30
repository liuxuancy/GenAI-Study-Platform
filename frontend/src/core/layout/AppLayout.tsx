// 登录后的页面框架：顶栏显示当前用户和退出按钮，左侧菜单，右侧是子路由页面。
import { Button, Layout, Menu, Tag } from 'antd'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import styles from '@/core/layout/AppLayout.module.css'

// 菜单按角色写死。阶段 7 再改成按权限收集。

function menuFor(role?: string) {
  if (role === 'ROLE_ADMIN') {
    return [
      { key: '/admin', label: 'Users' },
      { key: '/admin/classes', label: 'Classes' },
      { key: '/admin/courses', label: 'Courses' },
      { key: '/admin/system', label: 'System' },
      { key: '/account', label: 'Account' },
    ]
  }
  if (role === 'ROLE_TEACHER') {
    return [
      { key: '/teacher', label: 'Courses' },
      { key: '/account', label: 'Account' },
    ]
  }
  if (role === 'ROLE_STUDENT') {
    return [
      { key: '/student', label: 'Courses' },
      { key: '/account', label: 'Account' },
    ]
  }
  return [{ key: '/', label: 'Home' }]
}

export function AppLayout() {
  const currentUser = useAuthStore((state) => state.currentUser)
  const logout = useAuthStore((state) => state.logout)
  const navigate = useNavigate()
  const location = useLocation()
  const menuItems = menuFor(currentUser?.role)

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Layout.Header className={styles.header}>
        <div className={styles.brand}>
          <span className={styles.mark} aria-hidden="true">
            C
          </span>
          <span className={styles.brandName}>CaveClass</span>
        </div>
        <div className={styles.user}>
          <span>{currentUser?.nickname || currentUser?.username}</span>
          <Tag>{currentUser?.roleName}</Tag>
          <Button onClick={logout}>Sign out</Button>
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
