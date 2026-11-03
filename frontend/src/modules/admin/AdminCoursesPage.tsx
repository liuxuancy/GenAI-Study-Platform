// 管理员课程：全部课程，下架或恢复。不改正文，不改班级关系。
import { useEffect, useMemo, useState } from 'react'
import { Alert, Button, Input, Modal, Select, Space, Table, Tag, Typography, message } from 'antd'
import { Navigate } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import { getAdminCourses, setAdminCoursePublished, type AdminCourse } from '@/modules/admin/api'

export function AdminCoursesPage() {
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
      title: published ? `Restore "${course.title}"?` : `Remove "${course.title}" from students?`,
      content: published
        ? 'Students who are allowed to see this course will see it again. Class assignments stay as they are.'
        : 'Other students will no longer see it. The owner still can. Chapters, exercises, progress, and class assignments stay.',
      okText: published ? 'Restore' : 'Remove',
      okButtonProps: { danger: !published },
      onOk: async () => {
        try {
          await setAdminCoursePublished(course.id, published)
          await reload()
        } catch (error) {
          message.error(error instanceof Error ? error.message : 'Could not update the course.')
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
        Courses
      </Typography.Title>
      {courses ? (
        <Typography.Paragraph type="secondary">
          {publishedCount} published · {total - publishedCount} removed. Removing a course only hides it from other students.
        </Typography.Paragraph>
      ) : null}
      <Space wrap style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="Course or owner"
          value={keyword}
          onChange={(event) => {
            setKeyword(event.target.value)
            setPage(1)
          }}
          style={{ width: 240 }}
        />
        <Select
          allowClear
          placeholder="Owner role"
          value={ownerRole}
          onChange={(value) => {
            setOwnerRole(value)
            setPage(1)
          }}
          style={{ width: 140 }}
          options={[
            { value: 'ROLE_ADMIN', label: 'Admin' },
            { value: 'ROLE_TEACHER', label: 'Teacher' },
          ]}
        />
        <Select
          allowClear
          placeholder="Status"
          value={published}
          onChange={(value) => {
            setPublished(value)
            setPage(1)
          }}
          style={{ width: 140 }}
          options={[
            { value: true, label: 'Published' },
            { value: false, label: 'Removed' },
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
          Reset
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
          { title: 'Title', dataIndex: 'title' },
          {
            title: 'Owner',
            key: 'owner',
            render: (_, course) => course.ownerNickname || course.ownerUsername,
          },
          { title: 'Role', dataIndex: 'ownerRoleName' },
          {
            title: 'Classes',
            dataIndex: 'classCount',
            width: 100,
          },
          {
            title: 'Status',
            dataIndex: 'published',
            width: 130,
            render: (published: boolean) => (published ? <Tag>Published</Tag> : <Tag>Removed</Tag>),
          },
          {
            title: '',
            key: 'actions',
            width: 140,
            render: (_, course) => (
              <Space>
                {course.published ? (
                  <Button type="link" danger onClick={() => changePublished(course, false)}>
                    Remove
                  </Button>
                ) : (
                  <Button type="link" onClick={() => changePublished(course, true)}>
                    Restore
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
