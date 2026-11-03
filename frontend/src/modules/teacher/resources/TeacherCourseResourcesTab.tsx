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
import { useI18n } from '@/core/i18n/useI18n'
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
import { buildTeacherResourceScopeOptions } from './teacherResourceScopeOptions'
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

function formatDate(iso: string, dash: string): string {
  if (!iso) return dash
  const d = iso.slice(0, 10)
  return d || dash
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
  const { t } = useI18n()
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
      message.error(mapCourseResourceError(error, t('courseResources.teacherPanel.loadFailed')))
    } finally {
      setLoading(false)
    }
  }, [courseId, t])

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
      message.success(published ? t('courseResources.teacherPanel.publishedOn') : t('courseResources.teacherPanel.publishedOff'))
      void reload()
    } catch (error) {
      setAllItems((rows) =>
        rows.map((r) => (r.id === record.id ? { ...r, publishStatus: prev } : r)),
      )
      message.error(mapCourseResourceError(error, t('courseResources.teacherPanel.opFailed')))
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

  function rename(record: CourseResourceItem) {
    let nextName = record.displayName
    Modal.confirm({
      title: t('courseResources.teacherPanel.renameTitle'),
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
          message.warning(t('courseResources.teacherPanel.nameRequired'))
          return Promise.reject()
        }
        await patchTeacherCourseResource(courseId, record.id, { displayName: name })
        message.success(t('courseResources.teacherPanel.renamed'))
        void reload()
      },
    })
  }

  function changeScope(record: CourseResourceItem) {
    const initial =
      record.scopeType === 'course'
        ? 'course'
        : record.scopeType === 'chapter'
          ? `chapter:${record.chapterKey}`
          : `section:${record.chapterKey}:${record.sectionKey}`

    let scopeValue = initial

    const options = buildTeacherResourceScopeOptions(content, t)

    Modal.confirm({
      title: t('courseResources.teacherPanel.changeScopeTitle'),
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
        message.success(t('courseResources.teacherPanel.scopeUpdated'))
        void reload()
      },
    })
  }

  async function remove(record: CourseResourceItem) {
    Modal.confirm({
      title: t('courseResources.teacherPanel.deleteTitle'),
      content: t('courseResources.teacherPanel.deleteContent', { name: record.displayName }),
      okType: 'danger',
      okText: t('courseResources.teacherPanel.deleteBtn'),
      onOk: async () => {
        await deleteTeacherCourseResource(courseId, record.id)
        message.success(t('courseResources.teacherPanel.deleted'))
        void reload()
      },
    })
  }

  function rowMenu(record: CourseResourceItem): MenuProps['items'] {
    return [
      {
        key: 'rename',
        icon: <EditOutlined />,
        label: t('courseResources.teacherPanel.menuRename'),
        onClick: () => rename(record),
      },
      {
        key: 'scope',
        label: t('courseResources.teacherPanel.menuScope'),
        onClick: () => changeScope(record),
      },
      { type: 'divider' },
      {
        key: 'delete',
        icon: <DeleteOutlined />,
        label: t('courseResources.teacherPanel.menuDelete'),
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
          <div className={styles.eyebrow}>{t('courseResources.teacherPanel.eyebrow')}</div>
          <h2>{t('courseResources.teacherPanel.heroTitle')}</h2>
          <p className={styles.heroDesc}>
            {t('courseResources.teacherPanel.heroDescLine1')}
            <br />
            {t('courseResources.teacherPanel.heroDescLine2')}
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
            <span>{t('courseResources.teacherPanel.statTotal')}</span>
            <strong>{pad2(stats.total)}</strong>
          </div>
          <span className={styles.statSymbol}>▤</span>
        </div>
        <div className={styles.statCard}>
          <div>
            <span>{t('courseResources.teacherPanel.statPublished')}</span>
            <strong>{pad2(stats.published)}</strong>
          </div>
          <span className={styles.statSymbol}>◉</span>
        </div>
        <div className={styles.statCard}>
          <div>
            <span>{t('courseResources.teacherPanel.statDraft')}</span>
            <strong>{pad2(stats.draft)}</strong>
          </div>
          <span className={styles.statSymbol}>◌</span>
        </div>
      </div>

      <div className={styles.toolbar}>
        <div className={styles.chips}>
          {(
            [
              ['all', t('courseResources.teacherPanel.filterAll')],
              ['published', t('courseResources.teacherPanel.filterPublished')],
              ['draft', t('courseResources.teacherPanel.filterDraft')],
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
              { value: 'all', label: t('courseResources.studentPanel.allTypes') },
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
              { value: 'all', label: t('courseResources.studentPanel.allChapters') },
              ...(content?.chapters ?? []).map((ch, i) => ({
                value: ch.key,
                label: `${String(i + 1).padStart(2, '0')} · ${ch.title}`,
              })),
            ]}
          />
          <Input
            className={styles.search}
            placeholder={t('courseResources.teacherPanel.searchPlaceholder')}
            allowClear
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Button type="primary" onClick={() => setUploadOpen(true)}>
            {t('courseResources.teacherPanel.uploadBtn')}
          </Button>
        </div>
      </div>

      <div className={styles.listbox}>
        <div className={styles.listhead}>
          <span>{t('courseResources.teacherPanel.colFileName')}</span>
          <span>{t('courseResources.teacherPanel.colScope')}</span>
          <span>{t('courseResources.teacherPanel.colSize')}</span>
          <span>{t('courseResources.teacherPanel.colDate')}</span>
          <span>{t('courseResources.teacherPanel.colStatus')}</span>
          <span style={{ textAlign: 'right' }}>{t('courseResources.teacherPanel.colActions')}</span>
        </div>
        {loading ? (
          <div className={styles.loadingWrap}>
            <Spin />
          </div>
        ) : items.length === 0 ? (
          <div className={styles.empty}>{t('courseResources.teacherPanel.emptyList')}</div>
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
                    {record.fileExtension.toUpperCase()} · {t('courseResources.teacherPanel.uploadedByTeacher')}
                  </div>
                </div>
              </div>
              <div className={styles.meta}>{scopeLabel(record, content)}</div>
              <div className={styles.meta}>{formatFileSize(record.fileSize)}</div>
              <div className={styles.meta}>{formatDate(record.createdAt, t('common.dash'))}</div>
              <div className={styles.publishCell}>
                <span
                  className={`${styles.status} ${record.publishStatus === 'published' ? styles.statusPublished : ''}`}
                >
                  {record.publishStatus === 'published'
                    ? t('courseResources.teacherPanel.statusPublished')
                    : t('courseResources.teacherPanel.statusDraft')}
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
                  title={t('courseResources.teacherPanel.preview')}
                  onClick={() => void openPreview(record)}
                >
                  <EyeOutlined />
                </button>
                <button
                  type="button"
                  className={styles.iconBtn}
                  title={t('courseResources.teacherPanel.download')}
                  onClick={() => void download(record)}
                >
                  <DownloadOutlined />
                </button>
                <Dropdown menu={{ items: rowMenu(record) }} trigger={['click']}>
                  <button type="button" className={styles.iconBtn} title={t('courseResources.teacherPanel.more')}>
                    <MoreOutlined />
                  </button>
                </Dropdown>
              </div>
            </div>
          ))
        )}
      </div>

      <p className={styles.footnote}>{t('courseResources.teacherPanel.footnote')}</p>

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
