// 管理员课程：全部课程，下架或恢复。不改正文，不改班级关系。
import { useEffect, useMemo, useState } from 'react'
import { Alert, Button, Input, Modal, Select, Space, Table, Tag, Typography, message } from 'antd'
import { Navigate } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import { localizedRoleName } from '@/core/i18n/roleName'
import { useI18n } from '@/core/i18n/useI18n'
import { getAdminCourses, setAdminCoursePublished, type AdminCourse } from '@/modules/admin/api'

export function AdminCoursesPage() {
  const { t } = useI18n()
  const currentUser = useAuthStore((state) => state.currentUser)
  const [courses, setCourses] = useState<AdminCourse[] | null>(null)
  const [publishedCount, setPublishedCount] = useState(0)
  const [loadError, setLoadError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [published, setPublished] = useState<boolean>()
  const [ownerRole, setOwnerRole] = useState<string>()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  function reload() {
    return getAdminCourses()
      .then((result) => {
        setCourses(result.items)
        setPublishedCount(result.publishedCount)
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

  function changePublished(course: AdminCourse, published: boolean) {
    Modal.confirm({
      title: published
        ? t('admin.courses.restoreTitle', { title: course.title })
        : t('admin.courses.removeTitle', { title: course.title }),
      content: published ? t('admin.courses.restoreContent') : t('admin.courses.removeContent'),
      okText: published ? t('admin.courses.restore') : t('admin.courses.remove'),
      okButtonProps: { danger: !published },
      onOk: async () => {
        try {
          await setAdminCoursePublished(course.id, published)
          await reload()
        } catch (error) {
          message.error(error instanceof Error ? error.message : t('admin.courses.updateFailed'))
          throw error
        }
      },
    })
  }

  const total = courses?.length ?? 0
  const filteredCourses = useMemo(() => {
    const normalized = keyword.trim().toLowerCase()
    return (courses ?? []).filter((course) => {
      const textMatch =
        !normalized ||
        `${course.title} ${course.ownerUsername} ${course.ownerNickname}`.toLowerCase().includes(normalized)
      return textMatch &&
        (published === undefined || course.published === published) &&
        (!ownerRole || course.ownerRole === ownerRole)
    })
  }, [courses, keyword, ownerRole, published])

  return (
    <>
      <Typography.Title level={3} style={{ marginTop: 0 }}>
        {t('admin.courses.title')}
      </Typography.Title>
      {courses ? (
        <Typography.Paragraph type="secondary">
          {t('admin.courses.stats', {
            published: publishedCount,
            removed: total - publishedCount,
          })}
        </Typography.Paragraph>
      ) : null}
      <Space wrap style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder={t('admin.courses.searchPlaceholder')}
          value={keyword}
          onChange={(event) => {
            setKeyword(event.target.value)
            setPage(1)
          }}
          style={{ width: 240 }}
        />
        <Select
          allowClear
          placeholder={t('admin.courses.ownerRolePlaceholder')}
          value={ownerRole}
          onChange={(value) => {
            setOwnerRole(value)
            setPage(1)
          }}
          style={{ width: 140 }}
          options={(['ROLE_ADMIN', 'ROLE_TEACHER'] as const).map((value) => ({
            value,
            label: localizedRoleName(value),
          }))}
        />
        <Select
          allowClear
          placeholder={t('admin.courses.statusPlaceholder')}
          value={published}
          onChange={(value) => {
            setPublished(value)
            setPage(1)
          }}
          style={{ width: 140 }}
          options={[
            { value: true, label: t('admin.courses.published') },
            { value: false, label: t('admin.courses.removed') },
          ]}
        />
        <Button
          onClick={() => {
            setKeyword('')
            setOwnerRole(undefined)
            setPublished(undefined)
            setPage(1)
          }}
        >
          {t('admin.users.reset')}
        </Button>
      </Space>
      {loadError ? <Alert type="error" showIcon message={loadError} style={{ marginBottom: 16 }} /> : null}
      <Table<AdminCourse>
        rowKey="id"
        loading={courses === null && !loadError}
        dataSource={filteredCourses}
        pagination={{
          current: page,
          pageSize,
          total: filteredCourses.length,
          showSizeChanger: true,
          onChange: (nextPage, nextPageSize) => {
            setPage(nextPageSize === pageSize ? nextPage : 1)
            setPageSize(nextPageSize)
          },
        }}
        columns={[
          { title: t('admin.courses.colTitle'), dataIndex: 'title' },
          {
            title: t('admin.courses.colOwner'),
            key: 'owner',
            render: (_, course) => course.ownerNickname || course.ownerUsername,
          },
          { title: t('admin.courses.colRole'), dataIndex: 'ownerRoleName' },
          {
            title: t('admin.courses.colClasses'),
            dataIndex: 'classCount',
            width: 100,
          },
          {
            title: t('admin.courses.statusPlaceholder'),
            dataIndex: 'published',
            width: 130,
            render: (published: boolean) =>
              published ? <Tag>{t('admin.courses.published')}</Tag> : <Tag>{t('admin.courses.removed')}</Tag>,
          },
          {
            title: '',
            key: 'actions',
            width: 140,
            render: (_, course) => (
              <Space>
                {course.published ? (
                  <Button type="link" danger onClick={() => changePublished(course, false)}>
                    {t('admin.courses.remove')}
                  </Button>
                ) : (
                  <Button type="link" onClick={() => changePublished(course, true)}>
                    {t('admin.courses.restore')}
                  </Button>
                )}
              </Space>
            ),
          },
        ]}
      />
    </>
  )
}
