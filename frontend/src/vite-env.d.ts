/// <reference types="vite/client" />

declare module 'pdfjs-dist/build/pdf.worker.min.mjs' {
  export const WorkerMessageHandler: {
    setup: (handler: unknown, port: unknown) => void
  }
}
