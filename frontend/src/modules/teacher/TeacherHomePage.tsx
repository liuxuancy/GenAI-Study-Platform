// 讲师首页：课程列表、新建、勾选自己所在的班发布，以及这些班里的学生进度。
import { useEffect, useState } from 'react'
import { Alert, Button, Checkbox, Form, Input, Modal, Progress, Space, Table, Tag, Typography } from 'antd'
import { Navigate } from 'react-router-dom'

import { homePathFor } from '@/core/auth/homePath'
import { useAuthStore } from '@/core/auth/store'
import { ApiError } from '@/core/http'
import {
  createTeacherCourse,
  getTeacherClasses,
  generateTeacherCourse,
  getTeacherCourseProgress,
  getTeacherCourses,
  publishTeacherCourse,
  type ClassStudentProgress,
  type TeacherClass,
  type TeacherCourse,
} from '@/modules/teacher/api'

export function TeacherHomePage() {
  const currentUser = useAuthStore((state) => state.currentUser)
  const [courses, setCourses] = useState<TeacherCourse[] | null>(null)
  const [classes, setClasses] = useState<TeacherClass[]>([])
  const [loadError, setLoadError] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState('')
  const [publishCourse, setPublishCourse] = useState<TeacherCourse | null>(null)
  const [selectedClassIds, setSelectedClassIds] = useState<number[]>([])
  const [publishing, setPublishing] = useState(false)
  const [publishError, setPublishError] = useState('')
  const [progressCourse, setProgressCourse] = useState<TeacherCourse | null>(null)
  const [students, setStudents] = useState<ClassStudentProgress[]>([])
  const [progressError, setProgressError] = useState('')
  const [progressLoading, setProgressLoading] = useState(false)
  const [form] = Form.useForm<{ title: string; description?: string }>()
  const [generatingId, setGeneratingId] = useState<number | null>(null)
  const [generateError, setGenerateError] = useState('')

  useEffect(() => {
    if (currentUser?.role !== 'ROLE_TEACHER') {
      return
    }
    let cancelled = false
    Promise.all([getTeacherCourses(), getTeacherClasses()])
      .then(([courseItems, classItems]) => {
        if (!cancelled) {
          setCourses(courseItems)
          setClasses(classItems)
        }
      })
      .catch((error: Error) => {
        if (!cancelled) {
          setLoadError(error.message)
        }
      })
    return () => {
      cancelled = true
    }
  }, [currentUser?.role])

  if (!currentUser) {
    return null
  }
  if (currentUser.role !== 'ROLE_TEACHER') {
    return <Navigate to={homePathFor(currentUser.role)} replace />
  }

  const classNameById = new Map(classes.map((item) => [item.id, item.name]))

  function reloadCourses(nextProgress?: TeacherCourse) {
    return getTeacherCourses()
      .then((items) => {
        setCourses(items)
        if (nextProgress) {
          const updated = items.find((item) => item.id === nextProgress.id) ?? nextProgress
          setProgressCourse(updated)
        }
      })
      .catch((error: Error) => setLoadError(error.message))
  }

  function openPublish(course: TeacherCourse) {
    const known = new Set(classes.map((item) => item.id))
    setPublishCourse(course)
    setSelectedClassIds(course.classIds.filter((id) => known.has(id)))
    setPublishError('')
  }

  function openProgress(course: TeacherCourse) {
    setProgressCourse(course)
    setProgressError('')
    setProgressLoading(true)
    getTeacherCourseProgress(course.id)
      .then((result) => setStudents(result.students))
      .catch((error: Error) => setProgressError(error.message))
      .finally(() => setProgressLoading(false))
  }

  async function submitCreate() {
    const values = await form.validateFields()
    setCreating(true)
    setCreateError('')
    try {
      await createTeacherCourse({
        title: values.title.trim(),
        description: values.description?.trim() ?? '',
      })
      setCreateOpen(false)
      form.resetFields()
      await reloadCourses()
    } catch (error) {
      setCreateError(error instanceof ApiError ? error.message : 'Could not create the course.')
    } finally {
      setCreating(false)
    }
  }

  async function submitPublish() {
    if (!publishCourse) {
      return
    }
    setPublishing(true)
    setPublishError('')
    try {
      await publishTeacherCourse(publishCourse.id, selectedClassIds)
      const published = publishCourse
      setPublishCourse(null)
      await reloadCourses(progressCourse?.id === published.id ? published : undefined)
      if (progressCourse?.id === published.id) {
        openProgress({ ...published, classIds: selectedClassIds, published: selectedClassIds.length > 0 })
      }
    } catch (error) {
      setPublishError(error instanceof ApiError ? error.message : 'Could not update publication.')
    } finally {
      setPublishing(false)
    }
  }

  return (
    <>
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 16 }}>
        <Typography.Title level={3} style={{ margin: 0 }}>
          Courses
        </Typography.Title>
        <Button
          type="primary"
          onClick={() => {
            setCreateError('')
            form.resetFields()
            setCreateOpen(true)
          }}
        >
          New course
        </Button>
      </Space>
      {loadError ? <Alert type="error" showIcon message={loadError} style={{ marginBottom: 16 }} /> : null}
      {generateError ? <Alert type="error" showIcon message={generateError} style={{ marginBottom: 16 }} /> : null}
      <Table<TeacherCourse>
        rowKey="id"
        loading={courses === null && !loadError}
        dataSource={courses ?? []}
        pagination={false}
        locale={{ emptyText: 'No courses yet.' }}
        columns={[
          { title: 'Course', dataIndex: 'title' },
          {
            title: 'Description',
            dataIndex: 'description',
            ellipsis: true,
            render: (description: string) => description || '—',
          },
          {
            title: 'Status',
            dataIndex: 'published',
            width: 120,
            render: (published: boolean) =>
              published ? <Tag color="#111114">Published</Tag> : <Tag>Draft</Tag>,
          },
          {
            title: 'Classes',
            dataIndex: 'classIds',
            render: (classIds: number[]) =>
              classIds.length === 0 ? '—' : classIds.map((id) => classNameById.get(id) ?? `Class ${id}`).join(', '),
          },
          {
            title: '',
            key: 'actions',
            width: 300,
            render: (_, course) => (
              <Space>
                <Button
                  type="link"
                  disabled={generatingId === course.id}
                  onClick={() => {
                    setGeneratingId(course.id)
                    setGenerateError('')
                    generateTeacherCourse(course.id)
                      .then(() => reloadCourses())
                      .catch((error: Error) => setGenerateError(error.message))
                      .finally(() => setGeneratingId(null))
                  }}
                >
                  {generatingId === course.id ? 'Generating…' : 'Generate'}
                </Button>
                <Button type="link" onClick={() => openPublish(course)}>
                  Publish
                </Button>
                <Button type="link" onClick={() => openProgress(course)}>
                  Progress
                </Button>
              </Space>
            ),
          },
        ]}
      />

      {progressCourse ? (
        <div style={{ marginTop: 28 }}>
          <Typography.Title level={4}>Progress · {progressCourse.title}</Typography.Title>
          {progressCourse.published ? (
            <Typography.Paragraph type="secondary">
              Students in the classes this course is published to.
            </Typography.Paragraph>
          ) : (
            <Typography.Paragraph type="secondary">
              This course is not published, so there are no students to list.
            </Typography.Paragraph>
          )}
          {progressError ? <Alert type="error" showIcon message={progressError} style={{ marginBottom: 16 }} /> : null}
          <Table<ClassStudentProgress>
            rowKey="userId"
            loading={progressLoading}
            dataSource={students}
            pagination={false}
            locale={{ emptyText: 'No students.' }}
            columns={[
              {
                title: 'Student',
                dataIndex: 'nickname',
                render: (nickname: string, student) => nickname || student.username,
              },
              { title: 'Username', dataIndex: 'username' },
              {
                title: 'Progress',
                dataIndex: 'percent',
                width: 220,
                render: (percent: number) => <Progress percent={percent} />,
              },
            ]}
          />
        </div>
      ) : null}

      <Modal
        title="New course"
        open={createOpen}
        okText="Create"
        confirmLoading={creating}
        onCancel={() => setCreateOpen(false)}
        onOk={() => {
          void submitCreate()
        }}
      >
        {createError ? <Alert type="error" showIcon message={createError} style={{ marginBottom: 16 }} /> : null}
        <Form form={form} layout="vertical" requiredMark={false}>
          <Form.Item
            name="title"
            label="Course name"
            rules={[
              { required: true, whitespace: true, message: 'Enter a course name.' },
              { max: 200, message: 'Course name is too long.' },
            ]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="description" label="Description" rules={[{ max: 2000, message: 'Description is too long.' }]}>
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={publishCourse ? `Publish · ${publishCourse.title}` : 'Publish'}
        open={publishCourse !== null}
        okText="Save"
        confirmLoading={publishing}
        okButtonProps={{ disabled: classes.length === 0 }}
        onCancel={() => setPublishCourse(null)}
        onOk={() => {
          void submitPublish()
        }}
      >
        {publishError ? <Alert type="error" showIcon message={publishError} style={{ marginBottom: 16 }} /> : null}
        {classes.length === 0 ? (
          <Alert
            type="info"
            showIcon
            message="You are not in a class yet. An admin needs to add you before this course can be published."
          />
        ) : (
          <>
            <Typography.Paragraph type="secondary">
              Choose one or more of your classes. Leave them all unchecked to withdraw the course.
            </Typography.Paragraph>
            <Checkbox.Group
              style={{ display: 'flex', flexDirection: 'column', gap: 8 }}
              options={classes.map((item) => ({ label: item.name, value: item.id }))}
              value={selectedClassIds}
              onChange={(values) => setSelectedClassIds(values as number[])}
            />
          </>
        )}
      </Modal>
    </>
  )
}
