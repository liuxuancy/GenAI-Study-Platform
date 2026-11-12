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
import * as pdfjsWorker from 'pdfjs-dist/build/pdf.worker.min.mjs'

import type { ResourcePreviewResult } from '@/modules/courseResources/api'
import { useAuthStore } from '@/core/auth/store'
import { useI18n } from '@/core/i18n/useI18n'
import styles from './FilePreview.module.css'

/**
 * 静态打进主包，让 pdf.js 走主线程 fake worker。
 * 不要再用 ?url：线上会去拉 assets/pdf.worker*.mjs，动态 import 失败。
 */
const bundledWorker = pdfjsWorker as { WorkerMessageHandler?: unknown }
const pdfGlobal = globalThis as { pdfjsWorker?: { WorkerMessageHandler?: unknown } }
if (!pdfGlobal.pdfjsWorker?.WorkerMessageHandler && bundledWorker.WorkerMessageHandler) {
  pdfGlobal.pdfjsWorker = bundledWorker
}
pdfjs.GlobalWorkerOptions.workerSrc = ''

const PDF_LOG_PREFIX = '[FilePreview]'
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

function isPresignedCosUrl(url: string | undefined): url is string {
  return Boolean(url?.startsWith('http'))
}

/** 整包 GET → Blob → ArrayBuffer → Uint8Array；禁止 Range。 */
async function bytesFromResponse(response: Response, source: string): Promise<Uint8Array> {
  if (!response.ok) {
    throw new Error(`${source}_fetch_${response.status}`)
  }
  const blob = await response.blob()
  const arrayBuffer = await blob.arrayBuffer()
  // eslint-disable-next-line no-console -- 线上排障：确认 fetch 完整性与 pdfjs 入参大小
  console.info(PDF_LOG_PREFIX, 'fetch ok', {
    source,
    blobSize: blob.size,
    arrayBufferByteLength: arrayBuffer.byteLength,
  })
  return new Uint8Array(arrayBuffer)
}

async function fetchStreamPdfBytes(streamPath: string, token: string | null): Promise<Uint8Array> {
  const headers: Record<string, string> = {}
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  const response = await fetch(toAbsoluteUrl(streamPath), { credentials: 'omit', headers })
  return bytesFromResponse(response, 'stream')
}

async function fetchPresignedPdfBytes(cosUrl: string): Promise<Uint8Array> {
  const response = await fetch(cosUrl, { mode: 'cors', credentials: 'omit' })
  return bytesFromResponse(response, 'cos')
}

/** 优先同源 stream，COS 预签名为备选。绝不把 URL 传给 pdf.js。 */
async function loadPdfBytes(preview: ResourcePreviewResult, token: string | null): Promise<Uint8Array> {
  const attemptErrors: unknown[] = []

  if (preview.streamUrl) {
    try {
      return await fetchStreamPdfBytes(preview.streamUrl, token)
    } catch (err) {
      attemptErrors.push(err)
      console.warn(PDF_LOG_PREFIX, 'stream fetch failed, fallback to cos', err)
    }
  }

  const cosUrl = isPresignedCosUrl(preview.url) ? preview.url : ''
  if (cosUrl) {
    try {
      return await fetchPresignedPdfBytes(cosUrl)
    } catch (err) {
      attemptErrors.push(err)
      console.warn(PDF_LOG_PREFIX, 'cos fetch failed', err)
    }
  }

  const summary = attemptErrors.map((e) => (e instanceof Error ? e.message : String(e))).join('; ')
  throw new Error(`pdf_fetch_failed: ${summary || 'no_source'}`)
}

function openPdfDocument(bytes: Uint8Array): ReturnType<typeof pdfjs.getDocument> {
  const arrayBuffer = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)
  console.info(PDF_LOG_PREFIX, 'pdfjs.getDocument via data only', {
    byteLength: arrayBuffer.byteLength,
    disableRange: true,
  })
  return pdfjs.getDocument({
    data: new Uint8Array(arrayBuffer),
    disableRange: true,
    disableStream: true,
    disableAutoFetch: true,
  })
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

/**
 * 开发自测：控制台执行
 * `await window.__cavePdfPreviewSelfTest(previewJson)`
 * previewJson 为 /preview 接口返回的 data 对象。
 */
export async function cavePdfPreviewSelfTest(preview: ResourcePreviewResult): Promise<void> {
  const token = useAuthStore.getState().token
  const bytes = await loadPdfBytes(preview, token)
  const task = openPdfDocument(bytes)
  const doc = await task.promise
  console.info(PDF_LOG_PREFIX, 'selfTest ok', { numPages: doc.numPages, byteLength: bytes.byteLength })
  await doc.destroy()
}

if (import.meta.env.DEV) {
  ;(window as unknown as { __cavePdfPreviewSelfTest?: typeof cavePdfPreviewSelfTest }).__cavePdfPreviewSelfTest =
    cavePdfPreviewSelfTest
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
  const [pdfFallbackUrl, setPdfFallbackUrl] = useState('')
  const containerRef = useRef<HTMLDivElement>(null)
  const pdfDocRef = useRef<PDFDocumentProxy | null>(null)

  const presignedPdfUrl = isPresignedCosUrl(preview?.url) ? preview.url : undefined

  useEffect(() => {
    if (!open) {
      setPdfLoading(false)
      setPdfReady(false)
      setPdfError('')
      setImageError(false)
      setPdfPage(1)
      setPdfTotalPages(0)
      setPdfZoom(1)
      setPdfFallbackUrl((current) => {
        if (current) URL.revokeObjectURL(current)
        return ''
      })
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
    setPdfFallbackUrl((current) => {
      if (current) URL.revokeObjectURL(current)
      return ''
    })
    pdfDocRef.current = null

    const run = async () => {
      let bytes: Uint8Array | null = null
      try {
        const token = useAuthStore.getState().token
        bytes = await loadPdfBytes(preview, token)
        if (cancelled) return

        const loadingTask = openPdfDocument(bytes)
        const pdfDoc = await loadingTask.promise
        if (cancelled) {
          void pdfDoc.destroy()
          return
        }
        pdfDocRef.current = pdfDoc
        setPdfTotalPages(pdfDoc.numPages)
        setPdfReady(true)
      } catch (err) {
        console.error(PDF_LOG_PREFIX, 'pdfjs load failed', err)
        if (cancelled) return
        if (bytes && bytes.byteLength > 0) {
          const blobUrl = URL.createObjectURL(new Blob([bytes], { type: 'application/pdf' }))
          console.warn(PDF_LOG_PREFIX, 'fallback to native iframe via blob url', {
            byteLength: bytes.byteLength,
          })
          setPdfFallbackUrl(blobUrl)
          return
        }
        setPdfError(t('filePreview.pdfLoadFailed'))
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
      setPdfFallbackUrl((current) => {
        if (current) URL.revokeObjectURL(current)
        return ''
      })
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
    } catch (err) {
      console.error(PDF_LOG_PREFIX, 'page render failed', err)
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
  const showPdfToolbarDownload = isPdfPreview && (pdfReady || Boolean(pdfFallbackUrl)) && !pdfFailed

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
      if (pdfFallbackUrl) {
        return <iframe className={styles.frame} title={fileName} src={pdfFallbackUrl} />
      }
      if (pdfFailed) {
        return (
          <div className={styles.stateBlock}>
            <FileExclamationOutlined className={styles.stateIcon} />
            <p className={styles.message}>{pdfError}</p>
            {presignedPdfUrl ? (
              <Button
                type="primary"
                icon={<DownloadOutlined />}
                href={presignedPdfUrl}
                target="_blank"
                rel="noopener noreferrer"
              >
                {t('filePreview.downloadPdf')}
              </Button>
            ) : (
              <Button type="primary" icon={<DownloadOutlined />} onClick={onDownload}>
                {t('filePreview.downloadPdf')}
              </Button>
            )}
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
          {showPdfToolbarDownload || !isPdfPreview ? (
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
