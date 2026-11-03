// 管理员班级：新建班级，把教师和学生分进去。一个人可以在多个班。
import { useEffect, useMemo, useState } from 'react'
import { Alert, Button, Checkbox, Form, Input, Modal, Space, Table, Typography } from 'antd'
import { Navigate } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import {
  createAdminClass,
  getAdminClasses,
  getAdminUsers,
  renameAdminClass,
  setAdminClassMembers,
  type AdminClass,
  type AdminUser,
} from '@/modules/admin/api'

export function AdminClassesPage() {
  const currentUser = useAuthStore((state) => state.currentUser)
  const [classes, setClasses] = useState<AdminClass[] | null>(null)
  const [people, setPeople] = useState<AdminUser[]>([])
  const [keyword, setKeyword] = useState('')
  const [memberKeyword, setMemberKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [loadError, setLoadError] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [editing, setEditing] = useState<AdminClass | null>(null)
  const [memberIds, setMemberIds] = useState<number[]>([])
  const [formError, setFormError] = useState('')
  const [saving, setSaving] = useState(false)
  const [createForm] = Form.useForm<{ name: string }>()
  const [editForm] = Form.useForm<{ name: string }>()

  function reload() {
    return Promise.all([getAdminClasses(), getAdminUsers({ pageSize: 100 })])
      .then(([classItems, users]) => {
        setClasses(classItems)
        setPeople(users.items.filter((user) => user.role === 'ROLE_TEACHER' || user.role === 'ROLE_STUDENT'))
      })
      .catch((error: Error) => setLoadError(error.message))
  }

  useEffect(() => {
    if (currentUser?.role !== 'ROLE_ADMIN') {
      return
    }
    void reload()
  }, [currentUser?.role])

  if (!currentUser) {
    return null
  }
  if (currentUser.role !== 'ROLE_ADMIN') {
    return <Navigate to="/" replace />
  }

  async function createClass() {
    const values = await createForm.validateFields()
    setSaving(true)
    setFormError('')
    try {
      await createAdminClass(values.name.trim())
      setCreateOpen(false)
      createForm.resetFields()
      await reload()
    } catch (error) {
      setFormError(error instanceof Error ? error.message : 'Could not create the class.')
    } finally {
      setSaving(false)
    }
  }

  function openMembers(item: AdminClass) {
    setFormError('')
    setEditing(item)
    setMemberKeyword('')
    setMemberIds(item.members.map((member) => member.userId))
    editForm.setFieldsValue({ name: item.name })
  }

  const filteredClasses = useMemo(() => {
    const normalized = keyword.trim().toLowerCase()
    if (!normalized) {
      return classes ?? []
    }
    return (classes ?? []).filter(
      (item) =>
        item.name.toLowerCase().includes(normalized) ||
        item.members.some((member) => `${member.username} ${member.nickname}`.toLowerCase().includes(normalized)),
    )
  }, [classes, keyword])
  const visiblePeople = useMemo(() => {
    const normalized = memberKeyword.trim().toLowerCase()
    return normalized
      ? people.filter((person) => `${person.username} ${person.nickname}`.toLowerCase().includes(normalized))
      : people
  }, [people, memberKeyword])

  async function saveMembers() {
    if (!editing) {
      return
    }
    const values = await editForm.validateFields()
    setSaving(true)
    setFormError('')
    try {
      const name = values.name.trim()
      if (name !== editing.name) {
        await renameAdminClass(editing.id, name)
      }
      await setAdminClassMembers(editing.id, memberIds)
      setEditing(null)
      await reload()
    } catch (error) {
      setFormError(error instanceof Error ? error.message : 'Could not save the class.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 16 }}>
        <Typography.Title level={3} style={{ margin: 0 }}>
          Classes
        </Typography.Title>
        <Button
          type="primary"
          onClick={() => {
            setFormError('')
            createForm.resetFields()
            setCreateOpen(true)
          }}
        >
          New class
        </Button>
      </Space>
      <Input.Search
        allowClear
        placeholder="Search class or member"
        value={keyword}
        onChange={(event) => {
          setKeyword(event.target.value)
          setPage(1)
        }}
        style={{ width: 280, marginBottom: 16 }}
      />
      {loadError ? <Alert type="error" showIcon message={loadError} style={{ marginBottom: 16 }} /> : null}
      <Table<AdminClass>
        rowKey="id"
        loading={classes === null && !loadError}
        dataSource={filteredClasses}
        pagination={{
          current: page,
          pageSize,
          total: filteredClasses.length,
          showSizeChanger: true,
          onChange: (nextPage, nextPageSize) => {
            setPage(nextPageSize === pageSize ? nextPage : 1)
            setPageSize(nextPageSize)
          },
        }}
        locale={{ emptyText: 'No classes yet.' }}
        columns={[
          { title: 'Class', dataIndex: 'name' },
          {
            title: 'Members',
            dataIndex: 'members',
            render: (members: AdminClass['members']) =>
              members.length === 0
                ? '—'
                : members.map((member) => member.nickname || member.username).join(', '),
          },
          {
            title: '',
            key: 'actions',
            width: 120,
            render: (_, item) => (
              <Button type="link" onClick={() => openMembers(item)}>
                Members
              </Button>
            ),
          },
        ]}
      />

      <Modal
        title="New class"
        open={createOpen}
        okText="Create"
        confirmLoading={saving}
        onCancel={() => setCreateOpen(false)}
        onOk={() => void createClass()}
      >
        {formError ? <Alert type="error" showIcon message={formError} style={{ marginBottom: 16 }} /> : null}
        <Form form={createForm} layout="vertical">
          <Form.Item
            name="name"
            label="Class name"
            rules={[
              { required: true, whitespace: true, message: 'Enter a class name.' },
              { max: 64, message: 'Class name is too long.' },
            ]}
          >
            <Input />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editing ? `Members · ${editing.name}` : 'Members'}
        open={editing !== null}
        okText="Save"
        confirmLoading={saving}
        onCancel={() => setEditing(null)}
        onOk={() => void saveMembers()}
      >
        {formError ? <Alert type="error" showIcon message={formError} style={{ marginBottom: 16 }} /> : null}
        <Form form={editForm} layout="vertical">
          <Form.Item
            name="name"
            label="Class name"
            rules={[
              { required: true, whitespace: true, message: 'Enter a class name.' },
              { max: 64, message: 'Class name is too long.' },
            ]}
          >
            <Input />
          </Form.Item>
        </Form>
        <Typography.Paragraph type="secondary">Teachers and students. One person can belong to more than one class.</Typography.Paragraph>
        <Input.Search
          allowClear
          placeholder="Search people"
          value={memberKeyword}
          onChange={(event) => setMemberKeyword(event.target.value)}
          style={{ marginBottom: 12 }}
        />
        <Checkbox.Group
          style={{ display: 'flex', flexDirection: 'column', gap: 8 }}
          value={memberIds}
          options={visiblePeople.map((person) => ({
            label: `${person.nickname || person.username} (${person.role === 'ROLE_TEACHER' ? 'Teacher' : 'Student'})`,
            value: person.id,
          }))}
          onChange={(values) => setMemberIds(values as number[])}
        />
      </Modal>
    </>
  )
}
