// 我的账号：改昵称和邮箱，改自己的密码。用户名、学生专业、角色只读。
import { useState } from 'react'
import { Alert, Button, Form, Input, Typography } from 'antd'

import { changePassword, updateProfile } from '@/core/api/account'
import { useAuthStore } from '@/core/auth/store'
import { localizedRoleName } from '@/core/i18n/roleName'
import { useI18n } from '@/core/i18n/useI18n'

export function AccountPage() {
  const { t } = useI18n()
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
      setProfileError(error instanceof Error ? error.message : t('account.saveProfileFailed'))
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
      setPasswordError(error instanceof Error ? error.message : t('account.changePasswordFailed'))
      setSavingPassword(false)
    }
  }

  return (
    <>
      <Typography.Title level={3} style={{ marginTop: 0 }}>
        {t('account.title')}
      </Typography.Title>
      <Typography.Paragraph type="secondary">
        {t('account.roleHint', { role: localizedRoleName(currentUser.role) })}
      </Typography.Paragraph>
      {profileSaved ? (
        <Alert type="success" showIcon message={t('account.profileSaved')} style={{ marginBottom: 16 }} />
      ) : null}
      {profileError ? <Alert type="error" showIcon message={profileError} style={{ marginBottom: 16 }} /> : null}
      <Form
        form={profileForm}
        layout="vertical"
        style={{ maxWidth: 420 }}
        initialValues={{ nickname: currentUser.nickname, email: currentUser.email }}
        onFinish={() => void saveProfile()}
      >
        <Form.Item label={t('account.username')}>
          <Input value={currentUser.username} disabled />
        </Form.Item>
        <Form.Item
          name="nickname"
          label={t('account.nickname')}
          rules={[{ max: 64, message: t('account.nicknameTooLong') }]}
        >
          <Input />
        </Form.Item>
        <Form.Item
          name="email"
          label={t('account.email')}
          rules={[
            { type: 'email', message: t('account.emailInvalid') },
            { max: 128, message: t('account.emailTooLong') },
          ]}
        >
          <Input />
        </Form.Item>
        {currentUser.role === 'ROLE_STUDENT' ? (
          <Form.Item label={t('account.major')}>
            <Input value={currentUser.major || t('common.dash')} disabled />
          </Form.Item>
        ) : null}
        <Button type="primary" htmlType="submit" loading={savingProfile}>
          {t('account.saveProfile')}
        </Button>
      </Form>

      <Typography.Title level={4} style={{ marginTop: 36 }}>
        {t('account.passwordSection')}
      </Typography.Title>
      <Typography.Paragraph type="secondary">{t('account.passwordHint')}</Typography.Paragraph>
      {passwordError ? <Alert type="error" showIcon message={passwordError} style={{ marginBottom: 16 }} /> : null}
      <Form form={passwordForm} layout="vertical" style={{ maxWidth: 420 }} onFinish={() => void savePassword()}>
        <Form.Item
          name="oldPassword"
          label={t('account.currentPassword')}
          rules={[{ required: true, message: t('account.currentPasswordRequired') }]}
        >
          <Input.Password />
        </Form.Item>
        <Form.Item
          name="newPassword"
          label={t('account.newPassword')}
          rules={[
            { required: true, message: t('account.newPasswordRequired') },
            { min: 6, max: 72, message: t('account.passwordLength') },
          ]}
        >
          <Input.Password />
        </Form.Item>
        <Button type="primary" htmlType="submit" loading={savingPassword}>
          {t('account.changePassword')}
        </Button>
      </Form>
    </>
  )
}
