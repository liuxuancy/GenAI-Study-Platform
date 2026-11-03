// 管理员系统页：账号数、课程数、已发布数、全站 token，以及每位师生的用量明细。
import { useEffect, useMemo, useState } from 'react'
import { Alert, Button, Col, Input, Row, Select, Space, Statistic, Table, Tag, Typography } from 'antd'
import { Navigate, useSearchParams } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import { localizedRoleName } from '@/core/i18n/roleName'
import { useI18n } from '@/core/i18n/useI18n'
import {
  getAdminCourses,
  getAdminUsage,
  type AdminUsageCall,
  type AdminUsageReport,
  type AdminUsageUser,
} from '@/modules/admin/api'

const USAGE_SCENES = [
  'generate_course',
  'generate_activity',
  'evaluate_course',
  'chapter_qa',
  'learning_insight',
  'study_note',
] as const

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
  const { t } = useI18n()
  const currentUser = useAuthStore((state) => state.currentUser)
  const [searchParams, setSearchParams] = useSearchParams()
  const [report, setReport] = useState<AdminUsageReport | null>(null)
  const [courseTotal, setCourseTotal] = useState<number | null>(null)
  const [publishedCount, setPublishedCount] = useState<number | null>(null)
  const [loadError, setLoadError] = useState('')

  const sceneOptions = useMemo(
    () =>
      USAGE_SCENES.map((value) => ({
        value,
        label: t(`admin.system.scene.${value}`),
      })),
    [t],
  )

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

  function sceneLabel(scene: string) {
    const key = `admin.system.scene.${scene}`
    const translated = t(key)
    return translated === key ? scene : translated
  }

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
        {t('admin.system.title')}
      </Typography.Title>
      <Typography.Paragraph type="secondary">{t('admin.system.subtitle')}</Typography.Paragraph>
      {loadError ? <Alert type="error" showIcon message={loadError} style={{ marginBottom: 16 }} /> : null}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Statistic title={t('admin.system.accounts')} value={report?.accountCount ?? 0} loading={loading} />
        </Col>
        <Col span={6}>
          <Statistic
            title={t('admin.system.coursesStat')}
            value={courseTotal ?? 0}
            loading={courseTotal === null && !loadError}
          />
        </Col>
        <Col span={6}>
          <Statistic
            title={t('admin.system.publishedStat')}
            value={publishedCount ?? 0}
            loading={publishedCount === null && !loadError}
          />
        </Col>
        <Col span={6}>
          <Statistic title={t('admin.system.tokens')} value={report?.totalTokens ?? 0} loading={loading} />
          {report ? (
            <Typography.Text type="secondary">
              {t('admin.system.promptCompletion', {
                prompt: report.promptTokens,
                completion: report.completionTokens,
              })}
            </Typography.Text>
          ) : null}
        </Col>
      </Row>
      <Typography.Title level={4}>{t('admin.system.aiUsage')}</Typography.Title>
      <Space wrap style={{ marginBottom: 16 }}>
        <Button onClick={() => quickRange(1)}>{t('admin.system.today')}</Button>
        <Button onClick={() => quickRange(7)}>{t('admin.system.days7')}</Button>
        <Button onClick={() => quickRange(30)}>{t('admin.system.days30')}</Button>
        <Input
          type="date"
          aria-label={t('admin.system.fromDate')}
          value={searchParams.get('from') ?? ''}
          onChange={(event) => updateQuery({ from: event.target.value })}
          style={{ width: 145 }}
        />
        <Input
          type="date"
          aria-label={t('admin.system.toDate')}
          value={searchParams.get('to') ?? ''}
          onChange={(event) => updateQuery({ to: event.target.value })}
          style={{ width: 145 }}
        />
        <Input
          allowClear
          placeholder={t('admin.system.user')}
          value={searchParams.get('user') ?? ''}
          onChange={(event) => updateQuery({ user: event.target.value })}
          style={{ width: 150 }}
        />
        <Select
          allowClear
          placeholder={t('admin.system.role')}
          value={searchParams.get('role') || undefined}
          onChange={(value) => updateQuery({ role: value })}
          style={{ width: 130 }}
          options={(['ROLE_ADMIN', 'ROLE_TEACHER', 'ROLE_STUDENT'] as const).map((value) => ({
            value,
            label: localizedRoleName(value),
          }))}
        />
        <Select
          allowClear
          placeholder={t('admin.system.sceneLabel')}
          value={searchParams.get('scene') || undefined}
          onChange={(value) => updateQuery({ scene: value })}
          style={{ width: 180 }}
          options={sceneOptions}
        />
        <Input
          allowClear
          placeholder={t('admin.system.model')}
          value={searchParams.get('model') ?? ''}
          onChange={(event) => updateQuery({ model: event.target.value })}
          style={{ width: 150 }}
        />
        <Select
          allowClear
          placeholder={t('admin.system.status')}
          value={searchParams.get('call_status') || undefined}
          onChange={(value) => updateQuery({ call_status: value })}
          style={{ width: 130 }}
          options={[
            { value: 'success', label: t('admin.system.success') },
            { value: 'failed', label: t('admin.system.failed') },
          ]}
        />
        <Button onClick={() => setSearchParams({}, { replace: true })}>{t('admin.system.reset')}</Button>
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
                {
                  title: t('admin.system.colTime'),
                  dataIndex: 'createdAt',
                  render: (value: string) => formatWhen(value),
                },
                {
                  title: t('admin.system.colScene'),
                  dataIndex: 'scene',
                  render: (scene: string) => sceneLabel(scene),
                },
                {
                  title: t('admin.system.colModel'),
                  dataIndex: 'model',
                  render: (value: string) => value || t('common.dash'),
                },
                {
                  title: t('admin.system.colStatus'),
                  dataIndex: 'callStatus',
                  render: (status: string) =>
                    status === 'success' ? (
                      <Tag>{t('admin.system.success')}</Tag>
                    ) : (
                      <Tag>{t('admin.system.failed')}</Tag>
                    ),
                },
                { title: t('admin.system.colPrompt'), dataIndex: 'promptTokens' },
                { title: t('admin.system.colCompletion'), dataIndex: 'completionTokens' },
                { title: t('admin.system.colTotal'), dataIndex: 'totalTokens' },
              ]}
            />
          ),
        }}
        columns={[
          { title: t('admin.system.colUsername'), dataIndex: 'username' },
          {
            title: t('admin.system.colNickname'),
            dataIndex: 'nickname',
            render: (value: string) => value || t('common.dash'),
          },
          { title: t('admin.system.colRole'), dataIndex: 'roleName' },
          { title: t('admin.system.colCalls'), dataIndex: 'callCount' },
          { title: t('admin.system.colPrompt'), dataIndex: 'promptTokens' },
          { title: t('admin.system.colCompletion'), dataIndex: 'completionTokens' },
          { title: t('admin.system.colTotal'), dataIndex: 'totalTokens' },
        ]}
      />
    </>
  )
}
