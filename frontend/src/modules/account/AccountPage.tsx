// 我的账号：改昵称和邮箱，改自己的密码。学生专业只读。角色不可改。
import { useState } from 'react'
import { Alert, Button, Form, Input, Typography } from 'antd'

import { changePassword, updateProfile } from '@/core/api/account'
import { useAuthStore } from '@/core/auth/store'

export function AccountPage() {
  const currentUser = useAuthStore((state) => state.currentUser)
  const loadCurrentUser = useAuthStore((state) => state.loadCurrentUser)
  const clearSession = useAuthStore((state) => state.clearSession)
  const [profileError, setProfileError] = useState('')
  const [profileSaved, setProfileSaved] = useState(false)
  const [savingProfile, setSavingProfile] = useState(false)
  const [passwordError, setPasswordError] = useState('')
  const [savingPassword, setSavingPassword] = useState(false)
  const [profileForm] = Form.useForm<{ nickname: string; email: string }>()
  const [passwordForm] = Form.useForm<{ oldPassword: string; newPassword: string }>()

  if (!currentUser) {
    return null
  }

  async function saveProfile() {
    const values = await profileForm.validateFields()
    setSavingProfile(true)
    setProfileError('')
    setProfileSaved(false)
    try {
      await updateProfile({ nickname: values.nickname.trim(), email: values.email.trim() })
      await loadCurrentUser()
      setProfileSaved(true)
    } catch (error) {
      setProfileError(error instanceof Error ? error.message : 'Could not save your profile.')
    } finally {
      setSavingProfile(false)
    }
  }

  async function savePassword() {
    const values = await passwordForm.validateFields()
    setSavingPassword(true)
    setPasswordError('')
    try {
      await changePassword(values)
      clearSession()
    } catch (error) {
      setPasswordError(error instanceof Error ? error.message : 'Could not change your password.')
      setSavingPassword(false)
    }
  }

  return (
    <>
      <Typography.Title level={3} style={{ marginTop: 0 }}>
        Account
      </Typography.Title>
      <Typography.Paragraph type="secondary">
        {currentUser.username} · {currentUser.roleName}. Role cannot be changed.
      </Typography.Paragraph>
      {profileSaved ? <Alert type="success" showIcon message="Profile saved." style={{ marginBottom: 16 }} /> : null}
      {profileError ? <Alert type="error" showIcon message={profileError} style={{ marginBottom: 16 }} /> : null}
      <Form
        form={profileForm}
        layout="vertical"
        style={{ maxWidth: 420 }}
        initialValues={{ nickname: currentUser.nickname, email: currentUser.email }}
        onFinish={() => void saveProfile()}
      >
        <Form.Item name="nickname" label="Nickname" rules={[{ max: 64, message: 'Nickname is too long.' }]}>
          <Input />
        </Form.Item>
        <Form.Item
          name="email"
          label="Email"
          rules={[{ type: 'email', message: 'Enter a valid email.' }, { max: 128, message: 'Email is too long.' }]}
        >
          <Input />
        </Form.Item>
        {currentUser.role === 'ROLE_STUDENT' ? (
          <Form.Item label="Major">
            <Input value={currentUser.major || '—'} disabled />
          </Form.Item>
        ) : null}
        <Button type="primary" htmlType="submit" loading={savingProfile}>
          Save profile
        </Button>
      </Form>

      <Typography.Title level={4} style={{ marginTop: 36 }}>
        Password
      </Typography.Title>
      <Typography.Paragraph type="secondary">After this change you need to sign in again.</Typography.Paragraph>
      {passwordError ? <Alert type="error" showIcon message={passwordError} style={{ marginBottom: 16 }} /> : null}
      <Form form={passwordForm} layout="vertical" style={{ maxWidth: 420 }} onFinish={() => void savePassword()}>
        <Form.Item name="oldPassword" label="Current password" rules={[{ required: true, message: 'Enter your current password.' }]}>
          <Input.Password />
        </Form.Item>
        <Form.Item
          name="newPassword"
          label="New password"
          rules={[
            { required: true, message: 'Enter a new password.' },
            { min: 6, max: 72, message: 'Use 6 to 72 characters.' },
          ]}
        >
          <Input.Password />
        </Form.Item>
        <Button type="primary" htmlType="submit" loading={savingPassword}>
          Change password
        </Button>
      </Form>
    </>
  )
}

