import { useEffect, useMemo, useState } from 'react'
import { Form, Modal, Progress, Radio, Select, Upload, message } from 'antd'
import { InboxOutlined } from '@ant-design/icons'

import {
  publishTeacherCourseResource,
  uploadTeacherCourseResource,
} from '@/modules/courseResources/api'
import { mapCourseResourceError } from '@/modules/courseResources/errors'
import type { CourseContent } from '@/modules/teacher/api'
import styles from './teacherResources.module.css'

interface Props {
  open: boolean
  courseId: number
  content: CourseContent | null
  onClose: () => void
  onUploaded: () => void
}

interface QueueItem {
  uid: string
  file: File
  percent: number
  status: 'pending' | 'uploading' | 'done' | 'error'
  error?: string
}

function buildScopeOptions(content: CourseContent | null): { value: string; label: string }[] {
  const chapters = content?.chapters ?? []
  const options: { value: string; label: string }[] = [
    { value: 'course', label: '整门课程 · 公共资料' },
  ]
  chapters.forEach((ch, index) => {
    options.push({
      value: `chapter:${ch.key}`,
      label: `第 ${index + 1} 章 · ${ch.title}`,
    })
    ch.activities.forEach((act) => {
      options.push({
        value: `section:${ch.key}:${act.key}`,
        label: `${ch.title} · ${act.title}`,
      })
    })
  })
  return options
}

function parseScope(value: string): {
  scopeType: 'course' | 'chapter' | 'section'
  chapterKey: string
  sectionKey: string
} {
  if (value === 'course') {
    return { scopeType: 'course', chapterKey: '', sectionKey: '' }
  }
  if (value.startsWith('chapter:')) {
    return { scopeType: 'chapter', chapterKey: value.slice(8), sectionKey: '' }
  }
  if (value.startsWith('section:')) {
    const parts = value.split(':')
    return { scopeType: 'section', chapterKey: parts[1] ?? '', sectionKey: parts[2] ?? '' }
  }
  return { scopeType: 'course', chapterKey: '', sectionKey: '' }
}

export function TeacherResourceUploadModal({ open, courseId, content, onClose, onUploaded }: Props) {
  const [scopeValue, setScopeValue] = useState('course')
  const [publishOnUpload, setPublishOnUpload] = useState<'draft' | 'published'>('draft')
  const [queue, setQueue] = useState<QueueItem[]>([])
  const [submitting, setSubmitting] = useState(false)

  const scopeOptions = useMemo(() => buildScopeOptions(content), [content])

  useEffect(() => {
    if (!open) {
      setScopeValue('course')
      setPublishOnUpload('draft')
      setQueue([])
    }
  }, [open])

  async function uploadOne(item: QueueItem) {
    const { scopeType, chapterKey, sectionKey } = parseScope(scopeValue)
    if (scopeType === 'chapter' && !chapterKey) {
      throw new Error('请选择章节')
    }
    if (scopeType === 'section' && (!chapterKey || !sectionKey)) {
      throw new Error('请选择小节')
    }

    const form = new FormData()
    form.append('file', item.file)
    form.append('scopeType', scopeType)
    if (scopeType !== 'course') form.append('chapterKey', chapterKey)
    if (scopeType === 'section') form.append('sectionKey', sectionKey)

    setQueue((q) => q.map((row) => (row.uid === item.uid ? { ...row, status: 'uploading', percent: 0 } : row)))
    try {
      const created = await uploadTeacherCourseResource(courseId, form, (percent) => {
        setQueue((q) => q.map((row) => (row.uid === item.uid ? { ...row, percent } : row)))
      })
      if (publishOnUpload === 'published') {
        await publishTeacherCourseResource(courseId, created.id, true)
      }
      setQueue((q) => q.map((row) => (row.uid === item.uid ? { ...row, status: 'done', percent: 100 } : row)))
    } catch (error) {
      const msg = mapCourseResourceError(error, '上传失败')
      setQueue((q) =>
        q.map((row) => (row.uid === item.uid ? { ...row, status: 'error', error: msg } : row)),
      )
      throw error
    }
  }

  async function handleOk() {
    if (queue.length === 0) {
      message.warning('请先选择文件')
      return
    }
    const pending = queue.filter((row) => row.status !== 'done')
    if (pending.length === 0) {
      onUploaded()
      return
    }
    setSubmitting(true)
    try {
      for (const item of pending) {
        await uploadOne(item)
      }
      message.success('上传完成')
      setQueue([])
      onUploaded()
    } catch {
      message.error('部分文件上传失败，请查看下方错误说明后重试')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Modal
      title="上传课程资料"
      open={open}
      width={650}
      okText="开始上传"
      confirmLoading={submitting}
      onCancel={() => {
        if (!submitting) onClose()
      }}
      onOk={() => void handleOk()}
    >
      <Upload.Dragger
        multiple
        showUploadList={false}
        beforeUpload={(file) => {
          setQueue((q) => [...q, { uid: file.uid, file, percent: 0, status: 'pending' }])
          return false
        }}
        style={{ marginBottom: 12 }}
      >
        <p className="ant-upload-drag-icon">
          <InboxOutlined style={{ color: '#c84d38' }} />
        </p>
        <p className="ant-upload-text">拖拽文件到这里，或点击选择文件</p>
        <p className="ant-upload-hint">支持 PDF、PPTX、DOCX、XLSX、ZIP 等格式</p>
      </Upload.Dragger>

      {queue.length > 0 ? (
        <ul className={styles.queueList}>
          {queue.map((row) => (
            <li key={row.uid} className={styles.queueItem}>
              <div className={styles.queueHead}>
                <span className={styles.queueName} title={row.file.name}>
                  {row.file.name}
                </span>
                {row.status === 'uploading' ? (
                  <span className={styles.queuePct}>{row.percent}%</span>
                ) : null}
              </div>
              {row.status === 'uploading' || row.percent > 0 ? (
                <Progress
                  percent={row.percent}
                  size="small"
                  status={row.status === 'error' ? 'exception' : row.status === 'done' ? 'success' : 'active'}
                  showInfo={false}
                />
              ) : null}
              {row.error ? <p className={styles.queueError}>{row.error}</p> : null}
            </li>
          ))}
        </ul>
      ) : null}

      <Form layout="vertical" style={{ marginTop: 16 }}>
        <Form.Item label="资料所属位置">
          <Select value={scopeValue} onChange={setScopeValue} options={scopeOptions} />
        </Form.Item>
        <Form.Item label="发布设置">
          <Radio.Group
            value={publishOnUpload}
            onChange={(e) => setPublishOnUpload(e.target.value)}
          >
            <Radio value="draft">
              <strong>暂不发布</strong> · 仅教师可以查看和预览
            </Radio>
            <Radio value="published" style={{ display: 'block', marginTop: 8 }}>
              <strong>立即发布</strong> · 学生可以查看和下载
            </Radio>
          </Radio.Group>
        </Form.Item>
      </Form>
      <div
        style={{
          marginTop: 14,
          padding: '13px 15px',
          borderRadius: 10,
          background: '#fff7e8',
          color: '#8c6939',
          fontSize: 12,
          lineHeight: 1.7,
        }}
      >
        文件经后端鉴权后写入私有对象存储；上传失败不会写入课程资料列表。
      </div>
    </Modal>
  )
}
