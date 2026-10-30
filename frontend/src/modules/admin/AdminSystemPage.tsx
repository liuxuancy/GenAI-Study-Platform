// 管理员系统页：账号数、课程数、已发布数、全站 token，以及每位师生的用量明细。
import { useEffect, useState } from 'react'
import { Alert, Col, Row, Statistic, Table, Tag, Typography } from 'antd'
import { Navigate } from 'react-router-dom'

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
  study_note: 'Study note',
}

function formatWhen(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString()
}

export function AdminSystemPage() {
  const currentUser = useAuthStore((state) => state.currentUser)
  const [report, setReport] = useState<AdminUsageReport | null>(null)
  const [courseTotal, setCourseTotal] = useState<number | null>(null)
  const [publishedCount, setPublishedCount] = useState<number | null>(null)
  const [loadError, setLoadError] = useState('')

  useEffect(() => {
    if (currentUser?.role !== 'ROLE_ADMIN') {
      return
    }
    Promise.all([getAdminUsage(), getAdminCourses()])
      .then(([usage, courses]) => {
        setReport(usage)
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
      <Typography.Title level={4}>Teachers and students</Typography.Title>
      <Table<AdminUsageUser>
        rowKey="userId"
        loading={loading}
        dataSource={report?.users ?? []}
        pagination={false}
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
