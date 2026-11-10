import { useCallback, useEffect, useRef, useState } from 'react'
import { Button, Empty, Input, Progress, Select, Space, Table, Tag, message } from 'antd'

import { FilePreview } from '@/modules/shared/FilePreview/FilePreview'
import type { CourseContent } from '@/modules/teacher/api'
import {
  formatFileSize,
  getCourseResourceDownload,
  getCourseResourcePreview,
  listStudentCourseResources,
  uploadStudentCourseResource,
  scopeLabel,
  type CourseResourceItem,
} from '@/modules/courseResources/api'
import { mapCourseResourceError } from '@/modules/courseResources/errors'
import { useI18n } from '@/core/i18n/useI18n'
import styles from './studentResources.module.css'

interface Props {
  courseId: number
  content?: CourseContent | null
  canUpload?: boolean
}

export function StudentCourseResourcesPanel({ courseId, content = null, canUpload = false }: Props) {
  const { t } = useI18n()
  const [items, setItems] = useState<CourseResourceItem[]>([])
  const [loading, setLoading] = useState(false)
  const [typeFilter, setTypeFilter] = useState('all')
  const [chapterFilter, setChapterFilter] = useState('all')
  const [search, setSearch] = useState('')
  const [previewItem, setPreviewItem] = useState<CourseResourceItem | null>(null)
  const [preview, setPreview] = useState<Awaited<ReturnType<typeof getCourseResourcePreview>> | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewError, setPreviewError] = useState('')
  const [uploading, setUploading] = useState(false)
  const [uploadPercent, setUploadPercent] = useState(0)
  const fileInput = useRef<HTMLInputElement>(null)

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      const result = await listStudentCourseResources(courseId, {
        type: typeFilter === 'all' ? '' : typeFilter,
        chapterKey: chapterFilter === 'all' ? '' : chapterFilter,
        search: search.trim(),
      })
      setItems(result.items)
    } catch (error) {
      message.error(mapCourseResourceError(error, t('courseResources.studentPanel.loadFailed')))
    } finally {
      setLoading(false)
    }
  }, [chapterFilter, courseId, search, typeFilter])

  useEffect(() => {
    void reload()
  }, [reload])

  async function openPreview(record: CourseResourceItem) {
    setPreviewItem(record)
    setPreview(null)
    setPreviewError('')
    setPreviewLoading(true)
    try {
      setPreview(await getCourseResourcePreview(courseId, record.id))
    } catch (error) {
      setPreviewError(mapCourseResourceError(error, t('courseResources.studentPanel.previewFailed')))
    } finally {
      setPreviewLoading(false)
    }
  }

  async function download(record: CourseResourceItem) {
    try {
      const result = await getCourseResourceDownload(courseId, record.id)
      window.open(result.url, '_blank', 'noopener,noreferrer')
    } catch (error) {
      message.error(mapCourseResourceError(error, t('courseResources.studentPanel.downloadFailed')))
    }
  }

  async function upload(file: File) {
    setUploading(true)
    setUploadPercent(0)
    const form = new FormData()
    form.append('file', file)
    form.append('scopeType', 'course')
    try {
      await uploadStudentCourseResource(courseId, form, setUploadPercent)
      message.success(t('courseResources.uploadModal.uploadDone'))
      await reload()
    } catch (error) {
      message.error(mapCourseResourceError(error, t('courseResources.uploadModal.uploadFailed')))
    } finally {
      setUploading(false)
    }
  }

  if (!canUpload && !loading && items.length === 0 && !search && typeFilter === 'all' && chapterFilter === 'all') {
    return (
      <div className={styles.emptyWrap}>
        <Empty description={t('courseResources.studentPanel.notPublished')} />
      </div>
    )
  }

  return (
    <div className={styles.panel}>
      {canUpload ? (
        <div className={styles.uploadBar}>
          <input
            ref={fileInput}
            type="file"
            accept=".pdf,.ppt,.pptx,.doc,.docx,.xls,.xlsx,.zip,.rar,.7z,.png,.jpg,.jpeg,.gif,.webp,.txt,.md"
            hidden
            onChange={(event) => {
              const file = event.target.files?.[0]
              event.target.value = ''
              if (file) void upload(file)
            }}
          />
          <Button type="primary" loading={uploading} onClick={() => fileInput.current?.click()}>
            {t('courseResources.studentPanel.upload')}
          </Button>
          <span>{t('courseResources.studentPanel.uploadVisibility')}</span>
          {uploading ? <Progress percent={uploadPercent} size="small" /> : null}
        </div>
      ) : null}
      <div className={styles.filters}>
        <Select
          value={typeFilter}
          style={{ width: 120 }}
          onChange={setTypeFilter}
          options={[
            { value: 'all', label: t('courseResources.studentPanel.allTypes') },
            { value: 'pdf', label: 'PDF' },
            { value: 'pptx', label: 'PPTX' },
            { value: 'docx', label: 'DOCX' },
            { value: 'zip', label: 'ZIP' },
          ]}
        />
        <Select
          value={chapterFilter}
          style={{ width: 160 }}
          onChange={setChapterFilter}
          options={[
            { value: 'all', label: t('courseResources.studentPanel.allChapters') },
            ...(content?.chapters ?? []).map((ch: CourseContent['chapters'][number]) => ({
              value: ch.key,
              label: ch.title,
            })),
          ]}
        />
        <Input.Search placeholder={t('courseResources.studentPanel.searchPlaceholder')} allowClear onSearch={setSearch} style={{ width: 220 }} />
      </div>
      <div className={styles.tableWrap}>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={items}
        pagination={{ pageSize: 10 }}
        locale={{ emptyText: t('courseResources.studentPanel.emptyPublished') }}
        columns={[
          {
            title: t('courseResources.studentPanel.colName'),
            render: (_: unknown, r: CourseResourceItem) => (
              <Space>
                <Tag>{r.fileExtension.toUpperCase()}</Tag>
                {r.displayName}
              </Space>
            ),
          },
          { title: t('courseResources.studentPanel.colSize'), width: 100, render: (_: unknown, r: CourseResourceItem) => formatFileSize(r.fileSize) },
          { title: t('courseResources.studentPanel.colScope'), render: (_: unknown, r: CourseResourceItem) => scopeLabel(r, content) },
          {
            title: t('courseResources.studentPanel.colActions'),
            width: 160,
            render: (_: unknown, r: CourseResourceItem) => (
              <Space>
                <Button size="small" onClick={() => void openPreview(r)}>
                  {t('courseResources.studentPanel.preview')}
                </Button>
                <Button size="small" onClick={() => void download(r)}>
                  {t('courseResources.studentPanel.download')}
                </Button>
              </Space>
            ),
          },
        ]}
      />
      </div>
      <div className={styles.mobileList}>
        {loading ? (
          <p className={styles.mobileLoading}>{t('courseResources.studentPanel.loading')}</p>
        ) : items.length === 0 ? (
          <p className={styles.mobileEmpty}>{t('courseResources.studentPanel.emptyPublished')}</p>
        ) : (
          items.map((r) => (
          <div key={r.id} className={styles.mobileCard}>
            <div className={styles.mobileTitle}>
              <Tag>{r.fileExtension.toUpperCase()}</Tag>
              <span>{r.displayName}</span>
            </div>
            <div className={styles.mobileMeta}>
              {formatFileSize(r.fileSize)} · {scopeLabel(r, content)}
            </div>
            <div className={styles.mobileActions}>
              <Button size="small" block onClick={() => void openPreview(r)}>
                {t('courseResources.studentPanel.preview')}
              </Button>
              <Button size="small" block onClick={() => void download(r)}>
                {t('courseResources.studentPanel.download')}
              </Button>
            </div>
          </div>
        ))
        )}
      </div>
      <FilePreview
        open={Boolean(previewItem)}
        fileName={previewItem?.displayName ?? ''}
        fileType={previewItem?.fileExtension ?? ''}
        preview={preview}
        loading={previewLoading}
        error={previewError}
        onClose={() => setPreviewItem(null)}
        onDownload={() => previewItem && void download(previewItem)}
      />
    </div>
  )
}
