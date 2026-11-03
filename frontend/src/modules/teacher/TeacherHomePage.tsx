// 讲师首页：课程列表、新建、勾选自己所在的班发布，以及这些班里的学生进度。
import { useEffect, useState } from 'react'
import { Alert, Checkbox, Form, Input, InputNumber, Modal, Progress, Radio, Select, Table, Typography } from 'antd'
import { Navigate, useNavigate } from 'react-router-dom'

import { homePathFor } from '@/core/auth/homePath'
import { useAiProviderSelect } from '@/core/ai/useAiProviderSelect'
import type { AiGenerationMeta, GenerateCourseRequest } from '@/core/api/ai'
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
import styles from '@/modules/teacher/TeacherHomePage.module.css'

export function TeacherHomePage() {
  const currentUser = useAuthStore((state) => state.currentUser)
  const navigate = useNavigate()
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
  const [generateCourse, setGenerateCourse] = useState<TeacherCourse | null>(null)
  const [generateError, setGenerateError] = useState('')
  const [generateMeta, setGenerateMeta] = useState<AiGenerationMeta | null>(null)
  const [generateForm] = Form.useForm<GenerateCourseRequest>()
  const [generateMode, setGenerateMode] = useState<'outline' | 'full'>('outline')
  const {
    provider: aiProvider,
    selector: aiProviderSelector,
    error: aiProviderError,
    optionsAvailable: aiOptionsAvailable,
  } = useAiProviderSelect()

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

  const courseItems = courses ?? []
  const publishedCount = courseItems.filter((course) => course.published).length
  const draftCount = courseItems.length - publishedCount
  const needsAttention = courseItems.filter((course) => !course.hasContent || (course.published && course.classIds.length === 0)).length

  function openGeneration(course: TeacherCourse) {
    setGenerateCourse(course)
    setGenerateError('')
    setGenerateMeta(null)
    generateForm.setFieldsValue({
      topic: course.title,
      audience: course.audience || 'GENERAL',
      language: course.language || 'zh-CN',
      extraRequirements: '',
      chapterCountHint: course.chapterCount || undefined,
    })
  }

  async function startGeneration() {
    if (!generateCourse) return
    const values = await generateForm.validateFields()
    const course = generateCourse
    setGeneratingId(course.id)
    setGenerateError('')
    setGenerateMeta(null)
    try {
      const result = await generateTeacherCourse(course.id, {
        ...values,
        provider: aiProvider,
        outlineOnly: generateMode === 'outline',
      })
      setGenerateMeta(result.meta)
      await reloadCourses()
    } catch (error) {
      setGenerateError(error instanceof Error ? error.message : '课程生成失败。')
    } finally {
      setGeneratingId(null)
    }
  }

  return (
    <>
      <div className={styles.pageHeading}>
        <div>
          <p className={styles.eyebrow}>TEACHING STUDIO</p>
          <h1>课程中心</h1>
          <p>创建、完善并发布课程，集中查看班级学习进度。</p>
        </div>
        <button
          className={styles.primaryAction}
          type="button"
          onClick={() => {
            setCreateError('')
            form.resetFields()
            setCreateOpen(true)
          }}
        >
          创建课程
        </button>
      </div>

      <section className={styles.summaryGrid} aria-label="课程概览">
        <article><strong>{courseItems.length}</strong><span>全部课程</span></article>
        <article><strong>{draftCount}</strong><span>草稿</span></article>
        <article><strong>{publishedCount}</strong><span>已发布</span></article>
        <article><strong>{needsAttention}</strong><span>需要完善</span></article>
      </section>

      <div className={styles.modelBar}>
        <div><strong>AI 创作模型</strong><span>建议先生成课程大纲，进入工作台后逐节填充正文。</span></div>
        {aiProviderSelector}
      </div>
      {aiProviderError ? <Alert type="error" showIcon message={aiProviderError} className={styles.alert} /> : null}

      {loadError ? <Alert type="error" showIcon message={loadError} className={styles.alert} /> : null}
      {generateError ? <Alert type="error" showIcon message={generateError} className={styles.alert} /> : null}

      <section className={styles.courseSection}>
        <div className={styles.sectionHeading}>
          <div><p className={styles.eyebrow}>YOUR COURSES</p><h2>最近课程</h2></div>
          <span>{courseItems.length} 门课程</span>
        </div>
        {courses === null && !loadError ? (
          <div className={styles.skeletonGrid} aria-label="正在加载课程"><span /><span /><span /></div>
        ) : courseItems.length === 0 ? (
          <div className={styles.emptyState}><strong>还没有课程</strong><p>创建一门空白课程，再使用 AI 或手工方式完善内容。</p></div>
        ) : (
          <div className={styles.courseGrid}>
            {courseItems.map((course) => (
              <article className={styles.courseCard} key={course.id}>
                <div className={`${styles.cardArtwork} ${styles[`palette${course.id % 4}`] ?? ''}`}>
                  <span>{course.chapterCount || '—'}</span>
                  <small>{course.chapterCount ? 'CHAPTERS' : 'DRAFT'}</small>
                </div>
                <div className={styles.cardBody}>
                  <div className={styles.cardTags}>
                    <span className={course.published ? styles.published : undefined}>
                      {course.published ? '已发布' : '草稿'}
                    </span>
                    <span>{course.classIds.length ? `${course.classIds.length} 个班级` : '未选班级'}</span>
                  </div>
                  <h3>{course.title}</h3>
                  <p>{course.description || '暂无课程简介'}</p>
                  <div className={styles.classNames}>
                    {course.classIds.length
                      ? course.classIds.map((id) => classNameById.get(id) ?? `班级 ${id}`).join('、')
                      : course.hasContent ? '课程内容已准备，等待发布' : '需要生成或编写课程内容'}
                  </div>
                  <div className={styles.cardActions}>
                    <button type="button" onClick={() => navigate(`/teacher/courses/${course.id}`)}>进入工作台</button>
                    <button type="button" disabled={generatingId === course.id || !aiOptionsAvailable} onClick={() => openGeneration(course)}>
                      {generatingId === course.id ? '生成中…' : course.hasContent ? '重新生成' : 'AI 生成'}
                    </button>
                    <button type="button" onClick={() => openPublish(course)}>发布设置</button>
                    <button type="button" onClick={() => openProgress(course)}>学习进度</button>
                  </div>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>

      {progressCourse ? (
        <section className={styles.progressPanel}>
          <div className={styles.sectionHeading}>
            <div><p className={styles.eyebrow}>CLASS PROGRESS</p><h2>{progressCourse.title}</h2></div>
            <button type="button" onClick={() => setProgressCourse(null)}>关闭</button>
          </div>
          <Typography.Paragraph type="secondary">
            {progressCourse.published ? '以下为课程发布班级中的学生进度。' : '课程尚未发布，暂无可查看的学生。'}
          </Typography.Paragraph>
          {progressError ? <Alert type="error" showIcon message={progressError} style={{ marginBottom: 16 }} /> : null}
          <Table<ClassStudentProgress>
            rowKey="userId"
            loading={progressLoading}
            dataSource={students}
            pagination={false}
            locale={{ emptyText: '暂无学生' }}
            columns={[
              { title: '学生', dataIndex: 'nickname', render: (nickname: string, student) => nickname || student.username },
              { title: '用户名', dataIndex: 'username' },
              { title: '进度', dataIndex: 'percent', width: 220, render: (percent: number) => <Progress percent={percent} /> },
            ]}
          />
        </section>
      ) : null}

      <Modal
        title={generateCourse ? `AI 生成 · ${generateCourse.title}` : 'AI 生成课程'}
        open={generateCourse !== null}
        okText={generatingId ? '生成中…' : generateMode === 'outline' ? '生成大纲' : '生成完整课程'}
        confirmLoading={generatingId !== null}
        okButtonProps={{ disabled: !aiOptionsAvailable }}
        onCancel={() => { setGenerateCourse(null); setGenerateMode('outline') }}
        onOk={() => { void startGeneration() }}
      >
        {generateError ? <Alert type="error" showIcon message={generateError} style={{ marginBottom: 16 }} /> : null}
        {generateMeta ? <GenerationMeta meta={generateMeta} /> : null}
        <Typography.Paragraph type="secondary">
          生成只更新课程内容，不会发布课程。默认仅生成章与小节标题，正文在工作台「内容填充」阶段逐节生成。
        </Typography.Paragraph>
        <Radio.Group
          value={generateMode}
          onChange={(event) => setGenerateMode(event.target.value)}
          style={{ marginBottom: 16 }}
          optionType="button"
          options={[
            { label: '仅大纲（推荐）', value: 'outline' },
            { label: '完整课程（一次生成正文）', value: 'full' },
          ]}
        />
        <Form form={generateForm} layout="vertical">
          <Form.Item name="topic" label="主题" rules={[{ required: true, whitespace: true, message: '请输入课程主题' }]}>
            <Input maxLength={200} />
          </Form.Item>
          <Form.Item name="audience" label="受众">
            <Select options={[{ value: 'GENERAL', label: '通用' }, { value: 'CS', label: '计算机专业' }, { value: 'NON_CS', label: '非计算机专业' }]} />
          </Form.Item>
          <Form.Item name="language" label="语言">
            <Select options={[{ value: 'zh-CN', label: '简体中文' }, { value: 'en-US', label: 'English' }]} />
          </Form.Item>
          <Form.Item name="chapterCountHint" label="期望章节数">
            <InputNumber min={1} max={20} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="extraRequirements" label="补充要求">
            <Input.TextArea rows={3} maxLength={2000} showCount />
          </Form.Item>
        </Form>
      </Modal>

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
        okButtonProps={{
          disabled: classes.length === 0 || (!publishCourse?.published && selectedClassIds.length === 0),
        }}
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
              {publishCourse?.published
                ? '选择一个或多个班级；全部取消选择会撤回课程。'
                : '首次发布必须至少选择一个班级。'}
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

function GenerationMeta({ meta }: { meta: AiGenerationMeta }) {
  const details = [`耗时 ${meta.latencyMs} ms`]
  if (meta.modelUsed) details.push(`模型 ${meta.modelUsed}`)
  if (meta.fallbackUsed) details.push('已使用备用模型')
  return (
    <Alert
      type={meta.warnings.length || meta.fallbackUsed ? 'warning' : 'success'}
      showIcon
      message={details.join(' · ')}
      description={meta.warnings.length ? meta.warnings.join('；') : '内容生成完成'}
      style={{ marginBottom: 16 }}
    />
  )
}
