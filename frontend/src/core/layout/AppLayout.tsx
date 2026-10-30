// 登录后的页面框架：顶栏显示当前用户和退出按钮，左侧菜单，右侧是子路由页面。
import { Button, Layout, Menu, Tag } from 'antd'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import styles from '@/core/layout/AppLayout.module.css'

// 讲师多一个课程菜单。其余菜单仍按角色写在这里，阶段 7 再改成按权限收集。

export function AppLayout() {
  const currentUser = useAuthStore((state) => state.currentUser)
  const logout = useAuthStore((state) => state.logout)
  const navigate = useNavigate()
  const location = useLocation()
  const menuItems = [{ key: '/', label: 'Home' }]
  if (currentUser?.role === 'ROLE_TEACHER') {
    menuItems.push({ key: '/teacher', label: 'Courses' })
  }

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
