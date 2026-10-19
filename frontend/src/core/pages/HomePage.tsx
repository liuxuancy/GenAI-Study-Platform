// 首页：显示当前用户的角色和权限码，方便核对三个测试账号拿到的权限是否不同。
import { Card, Descriptions, Tag, Typography } from 'antd'

import { useAuthStore } from '@/core/auth/store'

export function HomePage() {
  const currentUser = useAuthStore((state) => state.currentUser)
  if (!currentUser) {
    return null
  }

  const permissionTags = currentUser.isAdmin ? (
    <Tag color="#c0442c">All permissions (admin)</Tag>
  ) : (
    currentUser.permissions.map((code) => <Tag key={code}>{code}</Tag>)
  )

  return (
    <Card>
      <Typography.Title level={3} style={{ marginTop: 0 }}>
        Welcome, {currentUser.nickname || currentUser.username}
      </Typography.Title>
      <Descriptions column={1} bordered size="small">
        <Descriptions.Item label="Username">{currentUser.username}</Descriptions.Item>
        <Descriptions.Item label="Role">
          {currentUser.roleName}（{currentUser.role}）
        </Descriptions.Item>
        <Descriptions.Item label="Permissions">{permissionTags}</Descriptions.Item>
      </Descriptions>
    </Card>
  )
}
