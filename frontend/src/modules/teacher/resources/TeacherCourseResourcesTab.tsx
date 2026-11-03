import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Button,
  Dropdown,
  Input,
  Modal,
  Select,
  Spin,
  Switch,
  message,
} from 'antd'
import type { MenuProps } from 'antd'
import {
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  EyeOutlined,
  MoreOutlined,
} from '@ant-design/icons'

import { FilePreview } from '@/modules/shared/FilePreview/FilePreview'
import type { CourseContent } from '@/modules/teacher/api'
import {
  deleteTeacherCourseResource,
  formatFileSize,
  getCourseResourceDownload,
  getCourseResourcePreview,
  listTeacherCourseResources,
  patchTeacherCourseResource,
  publishTeacherCourseResource,
  scopeLabel,
  type CourseResourceItem,
} from '@/modules/courseResources/api'
import { mapCourseResourceError } from '@/modules/courseResources/errors'
import { TeacherResourceUploadModal } from './TeacherResourceUploadModal'
import styles from './teacherResources.module.css'

interface Props {
  courseId: number
  content: CourseContent | null
}

type PublishFilter = 'all' | 'published' | 'draft'

function fileIconClass(ext: string): string {
  const e = ext.toLowerCase()
  if (e === 'pdf') return styles.fileIconPdf
  if (e === 'ppt' || e === 'pptx') return styles.fileIconPpt
  if (e === 'doc' || e === 'docx') return styles.fileIconDoc
  if (e === 'xls' || e === 'xlsx') return styles.fileIconXls
  if (e === 'zip' || e === 'rar' || e === '7z') return styles.fileIconZip
  return styles.fileIconDefault
}

function formatDate(iso: string): string {
  if (!iso) return '—'
  const d = iso.slice(0, 10)
  return d || '—'
}

function matchesType(item: CourseResourceItem, typeFilter: string): boolean {
  if (typeFilter === 'all') return true
  const ext = item.fileExtension.toLowerCase()
  if (typeFilter === 'pptx') return ext === 'ppt' || ext === 'pptx'
  if (typeFilter === 'docx') return ext === 'doc' || ext === 'docx'
  if (typeFilter === 'zip') return ext === 'zip' || ext === 'rar' || ext === '7z'
  return ext === typeFilter
}

export function TeacherCourseResourcesTab({ courseId, content }: Props) {
  const [allItems, setAllItems] = useState<CourseResourceItem[]>([])
  const [loading, setLoading] = useState(false)
  const [publishFilter, setPublishFilter] = useState<PublishFilter>('all')
  const [typeFilter, setTypeFilter] = useState('all')
  const [chapterFilter, setChapterFilter] = useState('all')
  const [search, setSearch] = useState('')
  const [uploadOpen, setUploadOpen] = useState(false)
  const [previewItem, setPreviewItem] = useState<CourseResourceItem | null>(null)
  const [preview, setPreview] = useState<Awaited<ReturnType<typeof getCourseResourcePreview>> | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewError, setPreviewError] = useState('')

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      const result = await listTeacherCourseResources(courseId, {})
      setAllItems(result.items)
    } catch (error) {
      message.error(mapCourseResourceError(error, '加载失败'))
    } finally {
      setLoading(false)
    }
  }, [courseId])

  useEffect(() => {
    void reload()
  }, [reload])

  const items = useMemo(() => {
    const q = search.trim().toLowerCase()
    return allItems.filter((item) => {
      if (publishFilter === 'published' && item.publishStatus !== 'published') return false
      if (publishFilter === 'draft' && item.publishStatus !== 'draft') return false
      if (chapterFilter !== 'all') {
        if (item.scopeType === 'course') return false
        if (item.chapterKey !== chapterFilter) return false
      }
      if (!matchesType(item, typeFilter)) return false
      if (q && !item.displayName.toLowerCase().includes(q) && !item.originalName.toLowerCase().includes(q)) {
        return false
      }
      return true
    })
  }, [allItems, chapterFilter, publishFilter, search, typeFilter])

  const stats = useMemo(() => {
    const published = allItems.filter((i) => i.publishStatus === 'published').length
    return { total: allItems.length, published, draft: allItems.length - published }
  }, [allItems])

  async function togglePublish(record: CourseResourceItem, published: boolean) {
    const prev = record.publishStatus
    setAllItems((rows) =>
      rows.map((r) => (r.id === record.id ? { ...r, publishStatus: published ? 'published' : 'draft' } : r)),
    )
    try {
      await publishTeacherCourseResource(courseId, record.id, published)
      message.success(published ? '已设为学生可见' : '已设为未发布')
      void reload()
    } catch (error) {
      setAllItems((rows) =>
        rows.map((r) => (r.id === record.id ? { ...r, publishStatus: prev } : r)),
      )
      message.error(mapCourseResourceError(error, '操作失败'))
    }
  }

  async function openPreview(record: CourseResourceItem) {
    setPreviewItem(record)
    setPreview(null)
    setPreviewError('')
    setPreviewLoading(true)
    try {
      const result = await getCourseResourcePreview(courseId, record.id)
      setPreview(result)
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

  function rename(record: CourseResourceItem) {
    let nextName = record.displayName
    Modal.confirm({
      title: '重命名资料',
      content: (
        <Input
          defaultValue={record.displayName}
          onChange={(e) => {
            nextName = e.target.value
          }}
        />
      ),
      onOk: async () => {
        const name = nextName.trim()
        if (!name) {
          message.warning('名称不能为空')
          return Promise.reject()
        }
        await patchTeacherCourseResource(courseId, record.id, { displayName: name })
        message.success('已重命名')
        void reload()
      },
    })
  }

  function changeScope(record: CourseResourceItem) {
    const chapters = content?.chapters ?? []
    const initial =
      record.scopeType === 'course'
        ? 'course'
        : record.scopeType === 'chapter'
          ? `chapter:${record.chapterKey}`
          : `section:${record.chapterKey}:${record.sectionKey}`

    let scopeValue = initial

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

    Modal.confirm({
      title: '调整归属位置',
      content: (
        <Select
          style={{ width: '100%' }}
          defaultValue={initial}
          options={options}
          onChange={(v) => {
            scopeValue = v
          }}
        />
      ),
      onOk: async () => {
        let body: Parameters<typeof patchTeacherCourseResource>[2]
        if (scopeValue === 'course') {
          body = { scopeType: 'course', chapterKey: '', sectionKey: '' }
        } else if (scopeValue.startsWith('chapter:')) {
          body = { scopeType: 'chapter', chapterKey: scopeValue.slice(8), sectionKey: '' }
        } else if (scopeValue.startsWith('section:')) {
          const [, chKey, secKey] = scopeValue.split(':')
          body = { scopeType: 'section', chapterKey: chKey, sectionKey: secKey }
        } else {
          return
        }
        await patchTeacherCourseResource(courseId, record.id, body)
        message.success('归属已更新')
        void reload()
      },
    })
  }

  async function remove(record: CourseResourceItem) {
    Modal.confirm({
      title: '删除资料',
      content: `确定删除「${record.displayName}」？文件将从存储中移除，此操作不可恢复。`,
      okType: 'danger',
      okText: '删除',
      onOk: async () => {
        await deleteTeacherCourseResource(courseId, record.id)
        message.success('已删除')
        void reload()
      },
    })
  }

  function rowMenu(record: CourseResourceItem): MenuProps['items'] {
    return [
      { key: 'rename', icon: <EditOutlined />, label: '重命名', onClick: () => rename(record) },
      { key: 'scope', label: '调整归属', onClick: () => changeScope(record) },
      { type: 'divider' },
      {
        key: 'delete',
        icon: <DeleteOutlined />,
        label: '删除',
        danger: true,
        onClick: () => void remove(record),
      },
    ]
  }

  const pad2 = (n: number) => String(n).padStart(2, '0')

  return (
    <div className={styles.page}>
      <div className={styles.hero}>
        <div>
          <div className={styles.eyebrow}>COURSE MATERIALS</div>
          <h2>课程资料管理</h2>
          <p className={styles.heroDesc}>
            上传和整理教学资源，自主决定向学生开放哪些内容。
            <br />
            支持课程、章节、小节三级关联，资料发布前可以随时预览。
          </p>
        </div>
        <div className={styles.heroGraphic} aria-hidden>
          <div className={styles.sheet} />
          <div className={styles.folder} />
        </div>
      </div>

      <div className={styles.stats}>
        <div className={styles.statCard}>
          <div>
            <span>全部资料</span>
            <strong>{pad2(stats.total)}</strong>
          </div>
          <span className={styles.statSymbol}>▤</span>
        </div>
        <div className={styles.statCard}>
          <div>
            <span>学生可见</span>
            <strong>{pad2(stats.published)}</strong>
          </div>
          <span className={styles.statSymbol}>◉</span>
        </div>
        <div className={styles.statCard}>
          <div>
            <span>未发布资料</span>
            <strong>{pad2(stats.draft)}</strong>
          </div>
          <span className={styles.statSymbol}>◌</span>
        </div>
      </div>

      <div className={styles.toolbar}>
        <div className={styles.chips}>
          {(
            [
              ['all', '全部资料'],
              ['published', '已发布'],
              ['draft', '未发布'],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              type="button"
              className={`${styles.chip} ${publishFilter === value ? styles.chipActive : ''}`}
              onClick={() => setPublishFilter(value)}
            >
              {label}
            </button>
          ))}
        </div>
        <div className={styles.tools}>
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
            style={{ width: 180 }}
            onChange={setChapterFilter}
            options={[
              { value: 'all', label: '全部章节' },
              ...(content?.chapters ?? []).map((ch, i) => ({
                value: ch.key,
                label: `${String(i + 1).padStart(2, '0')} · ${ch.title}`,
              })),
            ]}
          />
          <Input
            className={styles.search}
            placeholder="搜索文件名称…"
            allowClear
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Button type="primary" onClick={() => setUploadOpen(true)}>
            ＋ 上传资料
          </Button>
        </div>
      </div>

      <div className={styles.listbox}>
        <div className={styles.listhead}>
          <span>文件名称</span>
          <span>归属位置</span>
          <span>文件大小</span>
          <span>发布日期</span>
          <span>展示状态</span>
          <span style={{ textAlign: 'right' }}>操作</span>
        </div>
        {loading ? (
          <div className={styles.loadingWrap}>
            <Spin />
          </div>
        ) : items.length === 0 ? (
          <div className={styles.empty}>这里还没有符合条件的资料</div>
        ) : (
          items.map((record) => (
            <div key={record.id} className={styles.row}>
              <div className={styles.fileName}>
                <div className={`${styles.fileIcon} ${fileIconClass(record.fileExtension)}`}>
                  {record.fileExtension.slice(0, 4).toUpperCase()}
                </div>
                <div style={{ minWidth: 0 }}>
                  <div className={styles.fname} title={record.displayName}>
                    {record.displayName}
                  </div>
                  <div className={styles.sub}>
                    {record.fileExtension.toUpperCase()} · 教师上传
                  </div>
                </div>
              </div>
              <div className={styles.meta}>{scopeLabel(record, content)}</div>
              <div className={styles.meta}>{formatFileSize(record.fileSize)}</div>
              <div className={styles.meta}>{formatDate(record.createdAt)}</div>
              <div className={styles.publishCell}>
                <span
                  className={`${styles.status} ${record.publishStatus === 'published' ? styles.statusPublished : ''}`}
                >
                  {record.publishStatus === 'published' ? '● 学生可见' : '○ 未发布'}
                </span>
                <Switch
                  size="small"
                  checked={record.publishStatus === 'published'}
                  onChange={(checked) => void togglePublish(record, checked)}
                />
              </div>
              <div className={styles.actions}>
                <button
                  type="button"
                  className={styles.iconBtn}
                  title="预览"
                  onClick={() => void openPreview(record)}
                >
                  <EyeOutlined />
                </button>
                <button
                  type="button"
                  className={styles.iconBtn}
                  title="下载"
                  onClick={() => void download(record)}
                >
                  <DownloadOutlined />
                </button>
                <Dropdown menu={{ items: rowMenu(record) }} trigger={['click']}>
                  <button type="button" className={styles.iconBtn} title="更多">
                    <MoreOutlined />
                  </button>
                </Dropdown>
              </div>
            </div>
          ))
        )}
      </div>

      <p className={styles.footnote}>
        说明：发布状态保存在服务端；学生端仅能看到已发布资料。预览与下载经后端鉴权后签发短效链接。
      </p>

      <TeacherResourceUploadModal
        open={uploadOpen}
        courseId={courseId}
        content={content}
        onClose={() => setUploadOpen(false)}
        onUploaded={() => {
          setUploadOpen(false)
          void reload()
        }}
      />
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
