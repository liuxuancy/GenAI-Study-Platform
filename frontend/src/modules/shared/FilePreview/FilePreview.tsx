import { useEffect, useRef, useState } from 'react'
import { Button, Spin } from 'antd'
import { DownloadOutlined, FileExclamationOutlined } from '@ant-design/icons'
import * as pdfjs from 'pdfjs-dist'
import type { PDFDocumentProxy, PDFPageProxy } from 'pdfjs-dist'

import type { ResourcePreviewResult } from '@/modules/courseResources/api'
import styles from './FilePreview.module.css'

pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  'pdfjs-dist/build/pdf.worker.min.mjs',
  import.meta.url,
).toString()

const PDF_MAX_CSS_WIDTH = 900
const PDF_MAX_CSS_SCALE = 1.35
/** Cap DPR to limit memory on very dense displays */
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

const UNSUPPORTED_HINT = '当前文件暂不支持在线预览，请下载查看'

function outputScaleForCanvas(): number {
  return Math.min(window.devicePixelRatio || 1, PDF_MAX_OUTPUT_SCALE)
}

function cssScaleForPage(page: PDFPageProxy, containerWidth: number): number {
  const maxCssWidth = Math.min(PDF_MAX_CSS_WIDTH, containerWidth || PDF_MAX_CSS_WIDTH)
  const base = page.getViewport({ scale: 1 })
  if (base.width <= 0) return 1.15
  return Math.min(PDF_MAX_CSS_SCALE, maxCssWidth / base.width)
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
  const [pdfReady, setPdfReady] = useState(false)
  const [pdfError, setPdfError] = useState('')
  const [imageError, setImageError] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) {
      setPdfReady(false)
      setPdfError('')
      setImageError(false)
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
    if (!open || preview?.kind !== 'pdf' || !preview.url) {
      return
    }
    let cancelled = false
    let pdfDoc: PDFDocumentProxy | null = null
    let observer: IntersectionObserver | null = null

    setPdfReady(false)
    setPdfError('')

    const run = async () => {
      await new Promise<void>((resolve) => {
        requestAnimationFrame(() => resolve())
      })
      const host = containerRef.current
      if (!host || cancelled) return

      try {
        const loadingTask = pdfjs.getDocument({
          url: preview.url!,
          withCredentials: false,
          disableRange: false,
          disableStream: false,
          rangeChunkSize: 65536,
        })
        pdfDoc = await loadingTask.promise
        if (cancelled) return

        host.innerHTML = ''
        const containerWidth = host.clientWidth || PDF_MAX_CSS_WIDTH
        const numPages = pdfDoc.numPages

        const pageShells: { el: HTMLDivElement; pageNum: number; rendered: boolean }[] = []
        for (let pageNum = 1; pageNum <= numPages; pageNum += 1) {
          const wrap = document.createElement('div')
          wrap.className = styles.pdfPage
          wrap.dataset.page = String(pageNum)
          if (pageNum > 1) {
            wrap.classList.add(styles.pdfPagePending)
          }
          host.appendChild(wrap)
          pageShells.push({ el: wrap, pageNum, rendered: false })
        }

        const firstPage = await pdfDoc.getPage(1)
        if (cancelled) return
        const cssScale = cssScaleForPage(firstPage, containerWidth)

        if (pageShells[0]) {
          await renderPdfPageToHost(firstPage, pageShells[0].el, cssScale)
          pageShells[0].rendered = true
          pageShells[0].el.classList.remove(styles.pdfPagePending)
        }
        if (!cancelled) {
          setPdfReady(true)
        }

        const renderLazyPage = async (shell: (typeof pageShells)[number]) => {
          if (cancelled || shell.rendered || !pdfDoc) return
          shell.rendered = true
          try {
            const page = await pdfDoc.getPage(shell.pageNum)
            if (cancelled) return
            await renderPdfPageToHost(page, shell.el, cssScale)
            shell.el.classList.remove(styles.pdfPagePending)
          } catch {
            shell.el.classList.add(styles.pdfPageError)
            shell.el.replaceChildren(document.createTextNode('该页渲染失败'))
          }
        }

        if (numPages > 1) {
          const scrollRoot = host.closest(`.${styles.body}`)
          observer = new IntersectionObserver(
            (entries) => {
              for (const entry of entries) {
                if (!entry.isIntersecting) continue
                const el = entry.target as HTMLDivElement
                const pageNum = Number(el.dataset.page)
                const shell = pageShells.find((s) => s.pageNum === pageNum)
                if (!shell || shell.rendered) continue
                observer?.unobserve(el)
                void renderLazyPage(shell)
              }
            },
            { root: scrollRoot instanceof Element ? scrollRoot : null, rootMargin: '240px 0px' },
          )
          for (let i = 1; i < pageShells.length; i += 1) {
            observer.observe(pageShells[i].el)
          }
        }
      } catch {
        if (!cancelled) {
          setPdfError('PDF 加载失败，可能是链接已过期或浏览器无法访问存储地址，请下载查看')
          setPdfReady(false)
        }
      }
    }
    void run()
    return () => {
      cancelled = true
      observer?.disconnect()
      void pdfDoc?.destroy()
    }
  }, [open, preview?.kind, preview?.url])

  if (!open) {
    return null
  }

  const typeLabel = (fileType || 'file').toUpperCase()

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
            下载文件
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
            <p className={styles.message}>图片加载失败，请下载查看</p>
            <Button type="primary" onClick={onDownload}>
              下载
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
    if (preview?.kind === 'pdf' && preview.url) {
      if (pdfError) {
        return (
          <div className={styles.stateBlock}>
            <p className={styles.message}>{pdfError}</p>
            <Button type="primary" onClick={onDownload}>
              下载 PDF
            </Button>
          </div>
        )
      }
      return (
        <div className={styles.pdfWrap}>
          {!pdfReady ? <Spin className={styles.pdfSpinner} size="large" tip="正在加载 PDF…" /> : null}
          <div ref={containerRef} className={styles.pdfPages} aria-busy={!pdfReady} />
        </div>
      )
    }
    const unsupportedMessage = preview?.message?.trim() || UNSUPPORTED_HINT
    return (
      <div className={styles.stateBlock}>
        <FileExclamationOutlined className={styles.stateIcon} />
        <p className={styles.message}>{unsupportedMessage}</p>
        <Button type="primary" icon={<DownloadOutlined />} onClick={onDownload}>
          下载查看
        </Button>
      </div>
    )
  }

  return (
    <div className={styles.root} role="dialog" aria-modal="true" aria-label={`预览 ${fileName}`}>
      <header className={styles.toolbar}>
        <div className={styles.titleWrap}>
          <div className={styles.title}>{fileName}</div>
          <span className={styles.badge}>{typeLabel}</span>
        </div>
        <div className={styles.toolbarActions}>
          <Button icon={<DownloadOutlined />} onClick={onDownload}>
            下载
          </Button>
          <Button type="primary" onClick={onClose}>
            关闭
          </Button>
        </div>
      </header>
      <div className={styles.body}>{renderBody()}</div>
      <footer className={styles.footer}>
        预览与下载链接由服务端鉴权后短时签发，仅供本人学习使用，请勿对外转发。
      </footer>
    </div>
  )
}
