import { useCallback, useEffect, useState } from 'react'
import { Button, Empty, Input, Select, Space, Table, Tag, message } from 'antd'

import { FilePreview } from '@/modules/shared/FilePreview/FilePreview'
import type { CourseContent } from '@/modules/teacher/api'
import {
  formatFileSize,
  getCourseResourceDownload,
  getCourseResourcePreview,
  listStudentCourseResources,
  scopeLabel,
  type CourseResourceItem,
} from '@/modules/courseResources/api'
import { mapCourseResourceError } from '@/modules/courseResources/errors'
import styles from './studentResources.module.css'

interface Props {
  courseId: number
  content?: CourseContent | null
}

export function StudentCourseResourcesPanel({ courseId, content = null }: Props) {
  const [items, setItems] = useState<CourseResourceItem[]>([])
  const [loading, setLoading] = useState(false)
  const [typeFilter, setTypeFilter] = useState('all')
  const [chapterFilter, setChapterFilter] = useState('all')
  const [search, setSearch] = useState('')
  const [previewItem, setPreviewItem] = useState<CourseResourceItem | null>(null)
  const [preview, setPreview] = useState<Awaited<ReturnType<typeof getCourseResourcePreview>> | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewError, setPreviewError] = useState('')

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
      message.error(mapCourseResourceError(error, '加载失败'))
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
      setPreviewError(mapCourseResourceError(error, '预览失败'))
    } finally {
      setPreviewLoading(false)
    }
  }

  async function download(record: CourseResourceItem) {
    try {
      const result = await getCourseResourceDownload(courseId, record.id)
      window.open(result.url, '_blank', 'noopener,noreferrer')
    } catch (error) {
      message.error(mapCourseResourceError(error, '下载失败'))
    }
  }

  if (!loading && items.length === 0 && !search && typeFilter === 'all' && chapterFilter === 'all') {
    return (
      <div className={styles.emptyWrap}>
        <Empty description="教师暂未发布课程资料" />
      </div>
    )
  }

  return (
    <div className={styles.panel}>
      <div className={styles.filters}>
        <Select
          value={typeFilter}
          style={{ width: 120 }}
          onChange={setTypeFilter}
          options={[
            { value: 'all', label: '全部类型' },
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
            { value: 'all', label: '全部章节' },
            ...(content?.chapters ?? []).map((ch: CourseContent['chapters'][number]) => ({
              value: ch.key,
              label: ch.title,
            })),
          ]}
        />
        <Input.Search placeholder="搜索资料" allowClear onSearch={setSearch} style={{ width: 220 }} />
      </div>
      <div className={styles.tableWrap}>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={items}
        pagination={{ pageSize: 10 }}
        locale={{ emptyText: '暂无已发布资料' }}
        columns={[
          {
            title: '名称',
            render: (_: unknown, r: CourseResourceItem) => (
              <Space>
                <Tag>{r.fileExtension.toUpperCase()}</Tag>
                {r.displayName}
              </Space>
            ),
          },
          { title: '大小', width: 100, render: (_: unknown, r: CourseResourceItem) => formatFileSize(r.fileSize) },
          { title: '关联', render: (_: unknown, r: CourseResourceItem) => scopeLabel(r, content) },
          {
            title: '操作',
            width: 160,
            render: (_: unknown, r: CourseResourceItem) => (
              <Space>
                <Button size="small" onClick={() => void openPreview(r)}>
                  预览
                </Button>
                <Button size="small" onClick={() => void download(r)}>
                  下载
                </Button>
              </Space>
            ),
          },
        ]}
      />
      </div>
      <div className={styles.mobileList}>
        {loading ? (
          <p className={styles.mobileLoading}>加载中…</p>
        ) : items.length === 0 ? (
          <p className={styles.mobileEmpty}>暂无已发布资料</p>
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
                预览
              </Button>
              <Button size="small" block onClick={() => void download(r)}>
                下载
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
