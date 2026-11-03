export type AppLocale = 'zh' | 'en'

export type MessageTree = {
  [key: string]: string | MessageTree
}
