// 管理员用户列表：改昵称、邮箱、专业，启用或禁用，重置密码。角色不可改。
import { useEffect, useState } from 'react'
import { Alert, Button, Form, Input, Modal, Select, Space, Table, Tag, Typography } from 'antd'
import { Navigate } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import { localizedRoleName } from '@/core/i18n/roleName'
import { useI18n } from '@/core/i18n/useI18n'
import {
  getAdminUsers,
  resetAdminUserPassword,
  updateAdminUser,
  type AdminUser,
} from '@/modules/admin/api'

export function AdminUsersPage() {
  const { t } = useI18n()
  const currentUser = useAuthStore((state) => state.currentUser)
  const [users, setUsers] = useState<AdminUser[] | null>(null)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [role, setRole] = useState<string>()
  const [status, setStatus] = useState<number>()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [loadError, setLoadError] = useState('')
  const [editing, setEditing] = useState<AdminUser | null>(null)
  const [resetting, setResetting] = useState<AdminUser | null>(null)
  const [formError, setFormError] = useState('')
  const [saving, setSaving] = useState(false)
  const [editForm] = Form.useForm<{ nickname: string; email: string; major: string; status: number }>()
  const [passwordForm] = Form.useForm<{ password: string }>()

  function reload() {
    setLoadError('')
    return getAdminUsers({ keyword, role, status, page, pageSize })
      .then((result) => {
        setUsers(result.items)
        setTotal(result.total)
      })
      .catch((error: Error) => setLoadError(error.message))
  }

  useEffect(() => {
    if (currentUser?.role !== 'ROLE_ADMIN') {
      return
    }
    void reload()
  }, [currentUser?.role, keyword, role, status, page, pageSize])

  if (!currentUser) {
    return null
  }
  if (currentUser.role !== 'ROLE_ADMIN') {
    return <Navigate to="/" replace />
  }

  function openEdit(user: AdminUser) {
    setFormError('')
    setEditing(user)
    editForm.setFieldsValue({
      nickname: user.nickname,
      email: user.email,
      major: user.major,
      status: user.status,
    })
  }

  async function saveUser() {
    if (!editing) {
      return
    }
    const values = await editForm.validateFields()
    setSaving(true)
    setFormError('')
    try {
      await updateAdminUser(editing.id, {
        nickname: values.nickname.trim(),
        email: values.email.trim(),
        major: values.major.trim(),
        status: values.status,
      })
      setEditing(null)
      await reload()
    } catch (error) {
      setFormError(error instanceof Error ? error.message : t('admin.users.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  async function savePassword() {
    if (!resetting) {
      return
    }
    const values = await passwordForm.validateFields()
    setSaving(true)
    setFormError('')
    try {
      await resetAdminUserPassword(resetting.id, values.password)
      setResetting(null)
      passwordForm.resetFields()
    } catch (error) {
      setFormError(error instanceof Error ? error.message : t('admin.users.resetFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      <Typography.Title level={3} style={{ marginTop: 0 }}>
        {t('admin.users.title')}
      </Typography.Title>
      <Space wrap style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder={t('admin.users.searchPlaceholder')}
          value={keyword}
          onChange={(event) => {
            setKeyword(event.target.value)
            setPage(1)
          }}
          style={{ width: 240 }}
        />
        <Select
          allowClear
          placeholder={t('admin.users.rolePlaceholder')}
          value={role}
          onChange={(value) => {
            setRole(value)
            setPage(1)
          }}
          style={{ width: 140 }}
          options={(
            ['ROLE_ADMIN', 'ROLE_TEACHER', 'ROLE_STUDENT'] as const
          ).map((value) => ({ value, label: localizedRoleName(value) }))}
        />
        <Select
          allowClear
          placeholder={t('admin.users.statusPlaceholder')}
          value={status}
          onChange={(value) => {
            setStatus(value)
            setPage(1)
          }}
          style={{ width: 130 }}
          options={[
            { value: 1, label: t('admin.users.active') },
            { value: 0, label: t('admin.users.disabled') },
          ]}
        />
        <Button
          onClick={() => {
            setKeyword('')
            setRole(undefined)
            setStatus(undefined)
            setPage(1)
          }}
        >
          {t('admin.users.reset')}
        </Button>
      </Space>
      {loadError ? <Alert type="error" showIcon message={loadError} style={{ marginBottom: 16 }} /> : null}
      <Table<AdminUser>
        rowKey="id"
        loading={users === null && !loadError}
        dataSource={users ?? []}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          onChange: (nextPage, nextPageSize) => {
            setPage(nextPageSize === pageSize ? nextPage : 1)
            setPageSize(nextPageSize)
          },
        }}
        columns={[
          { title: t('admin.users.colUsername'), dataIndex: 'username' },
          {
            title: t('admin.users.colNickname'),
            dataIndex: 'nickname',
            render: (value: string) => value || t('common.dash'),
          },
          {
            title: t('admin.users.colEmail'),
            dataIndex: 'email',
            render: (value: string) => value || t('common.dash'),
          },
          {
            title: t('admin.users.colMajor'),
            dataIndex: 'major',
            render: (value: string) => value || t('common.dash'),
          },
          { title: t('admin.users.colRole'), dataIndex: 'roleName' },
          {
            title: t('admin.users.colStatus'),
            dataIndex: 'status',
            width: 110,
            render: (status: number) =>
              status === 1 ? <Tag>{t('admin.users.active')}</Tag> : <Tag>{t('admin.users.disabled')}</Tag>,
          },
          {
            title: '',
            key: 'actions',
            width: 180,
            render: (_, user) => (
              <Space>
                <Button type="link" onClick={() => openEdit(user)}>
                  {t('admin.users.edit')}
                </Button>
                <Button
                  type="link"
                  onClick={() => {
                    setFormError('')
                    passwordForm.resetFields()
                    setResetting(user)
                  }}
                >
                  {t('admin.users.resetPassword')}
                </Button>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        title={editing ? t('admin.users.editTitle', { username: editing.username }) : t('admin.users.edit')}
        open={editing !== null}
        okText={t('common.save')}
        confirmLoading={saving}
        onCancel={() => setEditing(null)}
        onOk={() => void saveUser()}
      >
        {formError ? <Alert type="error" showIcon message={formError} style={{ marginBottom: 16 }} /> : null}
        <Form form={editForm} layout="vertical">
          <Form.Item label={t('admin.users.role')}>
            <Input value={editing?.roleName} disabled />
          </Form.Item>
          <Form.Item
            name="nickname"
            label={t('admin.users.nickname')}
            rules={[{ max: 64, message: t('admin.users.nicknameTooLong') }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="email"
            label={t('admin.users.email')}
            rules={[{ type: 'email', message: t('admin.users.emailInvalid') }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="major"
            label={t('admin.users.major')}
            rules={[{ max: 64, message: t('admin.users.majorTooLong') }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="status" label={t('admin.users.colStatus')}>
            <Select
              disabled={editing?.id === currentUser.id}
              options={[
                { value: 1, label: t('admin.users.active') },
                { value: 0, label: t('admin.users.disabled') },
              ]}
            />
          </Form.Item>
          {editing?.id === currentUser.id ? (
            <Typography.Paragraph type="secondary">{t('admin.users.cannotDisableSelf')}</Typography.Paragraph>
          ) : null}
        </Form>
      </Modal>

      <Modal
        title={
          resetting
            ? t('admin.users.resetPasswordTitle', { username: resetting.username })
            : t('admin.users.resetPassword')
        }
        open={resetting !== null}
        okText={t('admin.users.resetBtn')}
        confirmLoading={saving}
        onCancel={() => setResetting(null)}
        onOk={() => void savePassword()}
      >
        {formError ? <Alert type="error" showIcon message={formError} style={{ marginBottom: 16 }} /> : null}
        <Form form={passwordForm} layout="vertical">
          <Form.Item
            name="password"
            label={t('admin.users.newPassword')}
            rules={[
              { required: true, message: t('admin.users.passwordRequired') },
              { min: 6, max: 72, message: t('admin.users.passwordLength') },
            ]}
          >
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}
