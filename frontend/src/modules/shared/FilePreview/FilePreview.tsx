import { useCallback, useEffect, useRef, useState } from 'react'
import { Button, Spin } from 'antd'
import {
  DownloadOutlined,
  FileExclamationOutlined,
  LeftOutlined,
  RightOutlined,
  ZoomInOutlined,
  ZoomOutOutlined,
} from '@ant-design/icons'
import * as pdfjs from 'pdfjs-dist'
import type { PDFDocumentProxy, PDFPageProxy } from 'pdfjs-dist'
import pdfWorkerSrc from 'pdfjs-dist/build/pdf.worker.min.mjs?url'

import type { ResourcePreviewResult } from '@/modules/courseResources/api'
import { useAuthStore } from '@/core/auth/store'
import { useI18n } from '@/core/i18n/useI18n'
import styles from './FilePreview.module.css'

/** Vite 生产构建需 ?url，否则 worker 404，pdf.js 会误报「链接过期」。 */
pdfjs.GlobalWorkerOptions.workerSrc = pdfWorkerSrc

const PDF_MAX_CSS_WIDTH = 900
const PDF_MAX_CSS_SCALE = 1.35
const PDF_MIN_ZOOM = 0.65
const PDF_MAX_ZOOM = 2.2
const PDF_ZOOM_STEP = 0.15
const PDF_MAX_OUTPUT_SCALE = 2.5

export interface FilePreviewProps {
  open: boolean
  fileName: string
  fileType: string
  preview: ResourcePreviewResult | null
  loading: boolean
  error: string
  onClose: () => void
  onDownload: () => void
}

function toAbsoluteUrl(pathOrUrl: string): string {
  try {
    return new URL(pathOrUrl, window.location.origin).href
  } catch {
    return pathOrUrl
  }
}

/** 整包 GET，不带 Range，避免预签名未签 Range 导致 403。 */
async function fetchPresignedPdfBytes(cosUrl: string): Promise<Uint8Array> {
  const response = await fetch(cosUrl, { mode: 'cors', credentials: 'omit' })
  if (!response.ok) {
    throw new Error(`cos_fetch_${response.status}`)
  }
  return new Uint8Array(await response.arrayBuffer())
}

async function fetchStreamPdfBytes(streamPath: string, token: string | null): Promise<Uint8Array> {
  const headers: Record<string, string> = {}
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  const response = await fetch(toAbsoluteUrl(streamPath), { credentials: 'omit', headers })
  if (!response.ok) {
    throw new Error(`stream_fetch_${response.status}`)
  }
  return new Uint8Array(await response.arrayBuffer())
}

async function loadPdfBytes(preview: ResourcePreviewResult, token: string | null): Promise<Uint8Array> {
  const cosUrl = preview.url?.startsWith('http') ? preview.url : ''
  if (cosUrl) {
    try {
      return await fetchPresignedPdfBytes(cosUrl)
    } catch {
      /* 香港桶常见 Range/CORS 403，走同源中转 */
    }
  }
  if (preview.streamUrl) {
    return fetchStreamPdfBytes(preview.streamUrl, token)
  }
  throw new Error('pdf_source_missing')
}

function outputScaleForCanvas(): number {
  return Math.min(window.devicePixelRatio || 1, PDF_MAX_OUTPUT_SCALE)
}

function cssScaleForPage(page: PDFPageProxy, containerWidth: number, zoom: number): number {
  const maxCssWidth = Math.min(PDF_MAX_CSS_WIDTH, containerWidth || PDF_MAX_CSS_WIDTH)
  const base = page.getViewport({ scale: 1 })
  if (base.width <= 0) return 1.15 * zoom
  return Math.min(PDF_MAX_CSS_SCALE, maxCssWidth / base.width) * zoom
}

async function renderPdfPageToHost(
  page: PDFPageProxy,
  hostEl: HTMLDivElement,
  cssScale: number,
): Promise<void> {
  const outputScale = outputScaleForCanvas()
  const viewport = page.getViewport({ scale: cssScale })
  const canvas = document.createElement('canvas')
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  canvas.width = Math.floor(viewport.width * outputScale)
  canvas.height = Math.floor(viewport.height * outputScale)
  canvas.style.width = `${Math.floor(viewport.width)}px`
  canvas.style.height = `${Math.floor(viewport.height)}px`

  const transform =
    outputScale !== 1 ? ([outputScale, 0, 0, outputScale, 0, 0] as [number, number, number, number, number, number]) : undefined

  hostEl.replaceChildren(canvas)
  await page.render({
    canvasContext: ctx,
    viewport,
    transform,
  }).promise
}

export function FilePreview({
  open,
  fileName,
  fileType,
  preview,
  loading,
  error,
  onClose,
  onDownload,
}: FilePreviewProps) {
  const { t } = useI18n()
  const [pdfLoading, setPdfLoading] = useState(false)
  const [pdfReady, setPdfReady] = useState(false)
  const [pdfError, setPdfError] = useState('')
  const [imageError, setImageError] = useState(false)
  const [pdfPage, setPdfPage] = useState(1)
  const [pdfTotalPages, setPdfTotalPages] = useState(0)
  const [pdfZoom, setPdfZoom] = useState(1)
  const containerRef = useRef<HTMLDivElement>(null)
  const pdfDocRef = useRef<PDFDocumentProxy | null>(null)

  useEffect(() => {
    if (!open) {
      setPdfLoading(false)
      setPdfReady(false)
      setPdfError('')
      setImageError(false)
      setPdfPage(1)
      setPdfTotalPages(0)
      setPdfZoom(1)
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  useEffect(() => {
    if (!open || preview?.kind !== 'pdf' || (!preview.url && !preview.streamUrl)) {
      return
    }
    let cancelled = false

    setPdfLoading(true)
    setPdfReady(false)
    setPdfError('')
    setPdfPage(1)
    pdfDocRef.current = null

    const run = async () => {
      try {
        const token = useAuthStore.getState().token
        const bytes = await loadPdfBytes(preview, token)
        if (cancelled) return
        // 复制一份交给 Worker，避免生产环境 SharedArrayBuffer / detach 问题；禁止走 URL Range 拉取。
        const loadingTask = pdfjs.getDocument({
          data: bytes.slice(0),
          disableRange: true,
          disableStream: true,
          disableAutoFetch: true,
          useSystemFonts: true,
        })
        const pdfDoc = await loadingTask.promise
        if (cancelled) {
          void pdfDoc.destroy()
          return
        }
        pdfDocRef.current = pdfDoc
        setPdfTotalPages(pdfDoc.numPages)
        setPdfReady(true)
      } catch {
        if (!cancelled) {
          setPdfError(t('filePreview.pdfLoadFailed'))
        }
      } finally {
        if (!cancelled) {
          setPdfLoading(false)
        }
      }
    }
    void run()
    return () => {
      cancelled = true
      void pdfDocRef.current?.destroy()
      pdfDocRef.current = null
    }
  }, [open, preview, t])

  const paintCurrentPage = useCallback(async () => {
    const host = containerRef.current
    const pdfDoc = pdfDocRef.current
    if (!host || !pdfDoc || !pdfReady) return
    try {
      const page = await pdfDoc.getPage(pdfPage)
      const containerWidth = host.clientWidth || PDF_MAX_CSS_WIDTH
      const cssScale = cssScaleForPage(page, containerWidth, pdfZoom)
      await renderPdfPageToHost(page, host, cssScale)
    } catch {
      setPdfError(t('filePreview.pageRenderFailed'))
    }
  }, [pdfPage, pdfReady, pdfZoom, t])

  useEffect(() => {
    if (!pdfReady) return
    void paintCurrentPage()
  }, [pdfReady, pdfPage, pdfZoom, paintCurrentPage])

  if (!open) {
    return null
  }

  const isPdfPreview = preview?.kind === 'pdf' && Boolean(preview.url || preview.streamUrl)
  const typeLabel = (fileType || 'file').toUpperCase()
  const pdfFailed = Boolean(pdfError)

  function renderPdfControls() {
    if (!pdfReady || pdfTotalPages < 1) return null
    return (
      <div className={styles.pdfControls}>
        <Button
          size="small"
          icon={<LeftOutlined />}
          disabled={pdfPage <= 1}
          onClick={() => setPdfPage((page) => Math.max(1, page - 1))}
        >
          {t('filePreview.prevPage')}
        </Button>
        <span className={styles.pdfPageLabel}>
          {t('filePreview.pageOf', { current: pdfPage, total: pdfTotalPages })}
        </span>
        <Button
          size="small"
          icon={<RightOutlined />}
          disabled={pdfPage >= pdfTotalPages}
          onClick={() => setPdfPage((page) => Math.min(pdfTotalPages, page + 1))}
        >
          {t('filePreview.nextPage')}
        </Button>
        <span className={styles.pdfControlDivider} aria-hidden />
        <Button
          size="small"
          icon={<ZoomOutOutlined />}
          disabled={pdfZoom <= PDF_MIN_ZOOM}
          onClick={() => setPdfZoom((z) => Math.max(PDF_MIN_ZOOM, Math.round((z - PDF_ZOOM_STEP) * 100) / 100))}
        />
        <span className={styles.pdfZoomLabel}>{Math.round(pdfZoom * 100)}%</span>
        <Button
          size="small"
          icon={<ZoomInOutlined />}
          disabled={pdfZoom >= PDF_MAX_ZOOM}
          onClick={() => setPdfZoom((z) => Math.min(PDF_MAX_ZOOM, Math.round((z + PDF_ZOOM_STEP) * 100) / 100))}
        />
        <Button size="small" onClick={() => setPdfZoom(1)}>
          {t('filePreview.zoomReset')}
        </Button>
      </div>
    )
  }

  function renderBody() {
    if (loading) {
      return <Spin size="large" />
    }
    if (error) {
      return (
        <div className={styles.stateBlock}>
          <FileExclamationOutlined className={styles.stateIcon} />
          <p className={styles.message}>{error}</p>
          <Button type="primary" icon={<DownloadOutlined />} onClick={onDownload}>
            {t('filePreview.downloadFile')}
          </Button>
        </div>
      )
    }
    if (preview?.kind === 'iframe' && preview.url) {
      return (
        <iframe
          className={styles.frame}
          title={fileName}
          src={preview.url}
          sandbox="allow-scripts allow-same-origin allow-popups"
        />
      )
    }
    if (preview?.kind === 'image' && preview.url) {
      if (imageError) {
        return (
          <div className={styles.stateBlock}>
            <p className={styles.message}>{t('filePreview.imageLoadFailed')}</p>
            <Button type="primary" onClick={onDownload}>
              {t('filePreview.download')}
            </Button>
          </div>
        )
      }
      return (
        <img
          className={styles.image}
          src={preview.url}
          alt={fileName}
          onError={() => setImageError(true)}
        />
      )
    }
    if (preview?.kind === 'pdf' && (preview.url || preview.streamUrl)) {
      if (pdfFailed) {
        return (
          <div className={styles.stateBlock}>
            <FileExclamationOutlined className={styles.stateIcon} />
            <p className={styles.message}>{pdfError}</p>
            <Button type="primary" icon={<DownloadOutlined />} onClick={onDownload}>
              {t('filePreview.downloadPdf')}
            </Button>
          </div>
        )
      }
      return (
        <div className={styles.pdfWrap}>
          {renderPdfControls()}
          {pdfLoading || !pdfReady ? (
            <Spin className={styles.pdfSpinner} size="large" tip={t('filePreview.loading')} />
          ) : null}
          <div
            ref={containerRef}
            className={`${styles.pdfPages}${pdfLoading ? ` ${styles.pdfPagesHidden}` : ''}`}
            aria-busy={pdfLoading}
          />
        </div>
      )
    }
    const unsupportedMessage = preview?.message?.trim() || t('filePreview.unsupported')
    return (
      <div className={styles.stateBlock}>
        <FileExclamationOutlined className={styles.stateIcon} />
        <p className={styles.message}>{unsupportedMessage}</p>
        <Button type="primary" icon={<DownloadOutlined />} onClick={onDownload}>
          {t('filePreview.downloadView')}
        </Button>
      </div>
    )
  }

  return (
    <div
      className={styles.root}
      role="dialog"
      aria-modal="true"
      data-file-preview-modal
      aria-label={t('filePreview.previewAria', { fileName })}
    >
      <header className={styles.toolbar}>
        <div className={styles.titleWrap}>
          <div className={styles.title}>{fileName}</div>
          <span className={styles.badge}>{typeLabel}</span>
        </div>
        <div className={styles.toolbarActions}>
          {!pdfFailed && !pdfLoading && isPdfPreview ? (
            <Button icon={<DownloadOutlined />} onClick={onDownload}>
              {t('filePreview.download')}
            </Button>
          ) : !isPdfPreview ? (
            <Button icon={<DownloadOutlined />} onClick={onDownload}>
              {t('filePreview.download')}
            </Button>
          ) : null}
          <Button type="primary" onClick={onClose}>
            {t('filePreview.close')}
          </Button>
        </div>
      </header>
      <div className={`${styles.body}${isPdfPreview ? ` ${styles.bodyScrollStart}` : ''}`}>
        {renderBody()}
      </div>
      <footer className={styles.footer}>{t('filePreview.footerNotice')}</footer>
    </div>
  )
}
