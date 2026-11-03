// 管理员用户列表：改昵称、邮箱、专业，启用或禁用，重置密码。角色不可改。
import { useEffect, useState } from 'react'
import { Alert, Button, Form, Input, Modal, Select, Space, Table, Tag, Typography } from 'antd'
import { Navigate } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import {
  getAdminUsers,
  resetAdminUserPassword,
  updateAdminUser,
  type AdminUser,
} from '@/modules/admin/api'

export function AdminUsersPage() {
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
      setFormError(error instanceof Error ? error.message : 'Could not save the user.')
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
      setFormError(error instanceof Error ? error.message : 'Could not reset the password.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      <Typography.Title level={3} style={{ marginTop: 0 }}>
        Users
      </Typography.Title>
      <Space wrap style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="Username or nickname"
          value={keyword}
          onChange={(event) => {
            setKeyword(event.target.value)
            setPage(1)
          }}
          style={{ width: 240 }}
        />
        <Select
          allowClear
          placeholder="Role"
          value={role}
          onChange={(value) => {
            setRole(value)
            setPage(1)
          }}
          style={{ width: 140 }}
          options={[
            { value: 'ROLE_ADMIN', label: 'Admin' },
            { value: 'ROLE_TEACHER', label: 'Teacher' },
            { value: 'ROLE_STUDENT', label: 'Student' },
          ]}
        />
        <Select
          allowClear
          placeholder="Status"
          value={status}
          onChange={(value) => {
            setStatus(value)
            setPage(1)
          }}
          style={{ width: 130 }}
          options={[
            { value: 1, label: 'Active' },
            { value: 0, label: 'Disabled' },
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
          Reset
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
          { title: 'Username', dataIndex: 'username' },
          { title: 'Nickname', dataIndex: 'nickname', render: (value: string) => value || '—' },
          { title: 'Email', dataIndex: 'email', render: (value: string) => value || '—' },
          { title: 'Major', dataIndex: 'major', render: (value: string) => value || '—' },
          { title: 'Role', dataIndex: 'roleName' },
          {
            title: 'Status',
            dataIndex: 'status',
            width: 110,
            render: (status: number) => (status === 1 ? <Tag>Active</Tag> : <Tag>Disabled</Tag>),
          },
          {
            title: '',
            key: 'actions',
            width: 180,
            render: (_, user) => (
              <Space>
                <Button type="link" onClick={() => openEdit(user)}>
                  Edit
                </Button>
                <Button
                  type="link"
                  onClick={() => {
                    setFormError('')
                    passwordForm.resetFields()
                    setResetting(user)
                  }}
                >
                  Reset password
                </Button>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        title={editing ? `Edit · ${editing.username}` : 'Edit'}
        open={editing !== null}
        okText="Save"
        confirmLoading={saving}
        onCancel={() => setEditing(null)}
        onOk={() => void saveUser()}
      >
        {formError ? <Alert type="error" showIcon message={formError} style={{ marginBottom: 16 }} /> : null}
        <Form form={editForm} layout="vertical">
          <Form.Item label="Role">
            <Input value={editing?.roleName} disabled />
          </Form.Item>
          <Form.Item name="nickname" label="Nickname" rules={[{ max: 64, message: 'Nickname is too long.' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="email" label="Email" rules={[{ type: 'email', message: 'Enter a valid email.' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="major" label="Major" rules={[{ max: 64, message: 'Major is too long.' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="status" label="Status">
            <Select
              disabled={editing?.id === currentUser.id}
              options={[
                { value: 1, label: 'Active' },
                { value: 0, label: 'Disabled' },
              ]}
            />
          </Form.Item>
          {editing?.id === currentUser.id ? (
            <Typography.Paragraph type="secondary">You cannot disable your own account.</Typography.Paragraph>
          ) : null}
        </Form>
      </Modal>

      <Modal
        title={resetting ? `Reset password · ${resetting.username}` : 'Reset password'}
        open={resetting !== null}
        okText="Reset"
        confirmLoading={saving}
        onCancel={() => setResetting(null)}
        onOk={() => void savePassword()}
      >
        {formError ? <Alert type="error" showIcon message={formError} style={{ marginBottom: 16 }} /> : null}
        <Form form={passwordForm} layout="vertical">
          <Form.Item
            name="password"
            label="New password"
            rules={[
              { required: true, message: 'Enter a password.' },
              { min: 6, max: 72, message: 'Use 6 to 72 characters.' },
            ]}
          >
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}
