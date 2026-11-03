// 管理员系统页：账号数、课程数、已发布数、全站 token，以及每位师生的用量明细。
import { useEffect, useState } from 'react'
import { Alert, Button, Col, Input, Row, Select, Space, Statistic, Table, Tag, Typography } from 'antd'
import { Navigate, useSearchParams } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import {
  getAdminCourses,
  getAdminUsage,
  type AdminUsageCall,
  type AdminUsageReport,
  type AdminUsageUser,
} from '@/modules/admin/api'

const sceneLabel: Record<string, string> = {
  generate_course: 'Generate course',
  generate_activity: 'Generate activity',
  evaluate_course: 'Evaluate course',
  chapter_qa: 'Chapter Q&A',
  learning_insight: 'Learning insight',
  study_note: 'Study note',
}

function formatWhen(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString()
}

function dateValue(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function AdminSystemPage() {
  const currentUser = useAuthStore((state) => state.currentUser)
  const [searchParams, setSearchParams] = useSearchParams()
  const [report, setReport] = useState<AdminUsageReport | null>(null)
  const [courseTotal, setCourseTotal] = useState<number | null>(null)
  const [publishedCount, setPublishedCount] = useState<number | null>(null)
  const [loadError, setLoadError] = useState('')

  const queryKey = searchParams.toString()

  useEffect(() => {
    if (currentUser?.role !== 'ROLE_ADMIN') {
      return
    }
    setLoadError('')
    setReport(null)
    getAdminUsage({
      from: searchParams.get('from') || undefined,
      to: searchParams.get('to') || undefined,
      user: searchParams.get('user') || undefined,
      role: searchParams.get('role') || undefined,
      scene: searchParams.get('scene') || undefined,
      model: searchParams.get('model') || undefined,
      call_status: searchParams.get('call_status') || undefined,
      page: Number(searchParams.get('page')) || 1,
      pageSize: Number(searchParams.get('pageSize')) || 20,
    })
      .then(setReport)
      .catch((error: Error) => setLoadError(error.message))
  }, [currentUser?.role, queryKey])

  useEffect(() => {
    if (currentUser?.role !== 'ROLE_ADMIN') {
      return
    }
    getAdminCourses()
      .then((courses) => {
        setCourseTotal(courses.total)
        setPublishedCount(courses.publishedCount)
      })
      .catch((error: Error) => setLoadError(error.message))
  }, [currentUser?.role])

  if (!currentUser) {
    return null
  }
  if (currentUser.role !== 'ROLE_ADMIN') {
    return <Navigate to="/" replace />
  }

  const loading = report === null && !loadError

  function updateQuery(changes: Record<string, string | number | undefined>, resetPage = true) {
    const next = new URLSearchParams(searchParams)
    Object.entries(changes).forEach(([key, value]) => {
      if (value === undefined || value === '') {
        next.delete(key)
      } else {
        next.set(key, String(value))
      }
    })
    if (resetPage) {
      next.delete('page')
    }
    setSearchParams(next, { replace: true })
  }

  function quickRange(days: number) {
    const to = new Date()
    const from = new Date(to)
    from.setDate(from.getDate() - days + 1)
    updateQuery({ from: dateValue(from), to: dateValue(to) })
  }

  return (
    <>
      <Typography.Title level={3} style={{ marginTop: 0 }}>
        System
      </Typography.Title>
      <Typography.Paragraph type="secondary">
        Token counts are model usage. They are not login credentials.
      </Typography.Paragraph>
      {loadError ? <Alert type="error" showIcon message={loadError} style={{ marginBottom: 16 }} /> : null}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Statistic title="Accounts" value={report?.accountCount ?? 0} loading={loading} />
        </Col>
        <Col span={6}>
          <Statistic title="Courses" value={courseTotal ?? 0} loading={courseTotal === null && !loadError} />
        </Col>
        <Col span={6}>
          <Statistic title="Published" value={publishedCount ?? 0} loading={publishedCount === null && !loadError} />
        </Col>
        <Col span={6}>
          <Statistic title="Tokens" value={report?.totalTokens ?? 0} loading={loading} />
          {report ? (
            <Typography.Text type="secondary">
              Prompt {report.promptTokens} · Completion {report.completionTokens}
            </Typography.Text>
          ) : null}
        </Col>
      </Row>
      <Typography.Title level={4}>AI usage</Typography.Title>
      <Space wrap style={{ marginBottom: 16 }}>
        <Button onClick={() => quickRange(1)}>Today</Button>
        <Button onClick={() => quickRange(7)}>7 days</Button>
        <Button onClick={() => quickRange(30)}>30 days</Button>
        <Input
          type="date"
          aria-label="From date"
          value={searchParams.get('from') ?? ''}
          onChange={(event) => updateQuery({ from: event.target.value })}
          style={{ width: 145 }}
        />
        <Input
          type="date"
          aria-label="To date"
          value={searchParams.get('to') ?? ''}
          onChange={(event) => updateQuery({ to: event.target.value })}
          style={{ width: 145 }}
        />
        <Input
          allowClear
          placeholder="User"
          value={searchParams.get('user') ?? ''}
          onChange={(event) => updateQuery({ user: event.target.value })}
          style={{ width: 150 }}
        />
        <Select
          allowClear
          placeholder="Role"
          value={searchParams.get('role') || undefined}
          onChange={(value) => updateQuery({ role: value })}
          style={{ width: 130 }}
          options={[
            { value: 'ROLE_ADMIN', label: 'Admin' },
            { value: 'ROLE_TEACHER', label: 'Teacher' },
            { value: 'ROLE_STUDENT', label: 'Student' },
          ]}
        />
        <Select
          allowClear
          placeholder="Scene"
          value={searchParams.get('scene') || undefined}
          onChange={(value) => updateQuery({ scene: value })}
          style={{ width: 180 }}
          options={Object.entries(sceneLabel).map(([value, label]) => ({ value, label }))}
        />
        <Input
          allowClear
          placeholder="Model"
          value={searchParams.get('model') ?? ''}
          onChange={(event) => updateQuery({ model: event.target.value })}
          style={{ width: 150 }}
        />
        <Select
          allowClear
          placeholder="Status"
          value={searchParams.get('call_status') || undefined}
          onChange={(value) => updateQuery({ call_status: value })}
          style={{ width: 130 }}
          options={[
            { value: 'success', label: 'Success' },
            { value: 'failed', label: 'Failed' },
          ]}
        />
        <Button onClick={() => setSearchParams({}, { replace: true })}>Reset</Button>
      </Space>
      <Table<AdminUsageUser>
        rowKey="userId"
        loading={loading}
        dataSource={report?.users ?? []}
        pagination={{
          current: report?.page ?? 1,
          pageSize: report?.pageSize ?? 20,
          total: report?.total ?? 0,
          showSizeChanger: true,
          onChange: (page, pageSize) => updateQuery({ page, pageSize }, false),
        }}
        expandable={{
          rowExpandable: (user) => user.calls.length > 0,
          expandedRowRender: (user) => (
            <Table<AdminUsageCall>
              rowKey="id"
              size="small"
              pagination={false}
              dataSource={user.calls}
              columns={[
                { title: 'Time', dataIndex: 'createdAt', render: (value: string) => formatWhen(value) },
                {
                  title: 'Scene',
                  dataIndex: 'scene',
                  render: (scene: string) => sceneLabel[scene] ?? scene,
                },
                { title: 'Model', dataIndex: 'model', render: (value: string) => value || '—' },
                {
                  title: 'Status',
                  dataIndex: 'callStatus',
                  render: (status: string) => (status === 'success' ? <Tag>Success</Tag> : <Tag>Failed</Tag>),
                },
                { title: 'Prompt', dataIndex: 'promptTokens' },
                { title: 'Completion', dataIndex: 'completionTokens' },
                { title: 'Total', dataIndex: 'totalTokens' },
              ]}
            />
          ),
        }}
        columns={[
          { title: 'Username', dataIndex: 'username' },
          { title: 'Nickname', dataIndex: 'nickname', render: (value: string) => value || '—' },
          { title: 'Role', dataIndex: 'roleName' },
          { title: 'Calls', dataIndex: 'callCount' },
          { title: 'Prompt', dataIndex: 'promptTokens' },
          { title: 'Completion', dataIndex: 'completionTokens' },
          { title: 'Total', dataIndex: 'totalTokens' },
        ]}
      />
    </>
  )
}
