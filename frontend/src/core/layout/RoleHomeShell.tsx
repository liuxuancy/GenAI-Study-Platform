// 角色首页外壳：左侧边栏、顶栏面包屑、内容区。学生、讲师、管理员共用，内容由调用方传入。
import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'

import { useI18n } from '@/core/i18n/useI18n'
import styles from '@/core/layout/RoleHomeShell.module.css'

export interface RoleNavItem {
  label: string
  active?: boolean
  onSelect?: () => void
}

export interface RoleHomeShellProps {
  roleLabel: string
  homePath: string
  navTitle: string
  navAriaLabel: string
  navItems: RoleNavItem[]
  pageLabel: string
  onSignOut: () => void
  contentFlush?: boolean
  children?: ReactNode
}

export function RoleHomeShell({
  roleLabel,
  homePath,
  navTitle,
  navAriaLabel,
  navItems,
  pageLabel,
  onSignOut,
  contentFlush = false,
  children,
}: RoleHomeShellProps) {
  const { t } = useI18n()
  return (
    <div className={`${styles.app} ${contentFlush ? styles.appFlush : ''}`}>
      <aside className={styles.side}>
        <div className={styles.sideBrand}>
          <Link className={styles.brand} to={homePath} aria-label={t('common.brandName')}>
            <span className={styles.mark} aria-hidden="true">
              C
            </span>
            <span className={styles.brandText}>
              <strong>CaveClass</strong>
              <small>{roleLabel}</small>
            </span>
          </Link>
        </div>

        <div className={styles.navGroup}>
          <p className={styles.navTitle}>{navTitle}</p>
          <nav className={styles.nav} aria-label={navAriaLabel}>
            {navItems.map((item) => (
              <a
                key={item.label}
                href="#"
                className={item.active ? styles.navOn : undefined}
                aria-current={item.active ? 'page' : undefined}
                onClick={(event) => {
                  event.preventDefault()
                  item.onSelect?.()
                }}
              >
                {item.label}
              </a>
            ))}
          </nav>
        </div>

        <div className={styles.sideFoot}>
          <button className={styles.signOut} type="button" onClick={onSignOut}>
            {t('common.signOut')}
          </button>
        </div>
      </aside>

      <div className={styles.main}>
        <header className={styles.appbar}>
          <div className={styles.crumbs}>
            <span>{roleLabel}</span>
            <span className={styles.sep}>/</span>
            <strong>{pageLabel}</strong>
          </div>
          <nav className={styles.mobileNav} aria-label={`${navAriaLabel} mobile`}>
            {navItems.map((item) => (
              <button
                key={item.label}
                type="button"
                className={item.active ? styles.mobileNavOn : undefined}
                onClick={item.onSelect}
              >
                {item.label}
              </button>
            ))}
            <button type="button" onClick={onSignOut}>{t('common.signOut')}</button>
          </nav>
        </header>

        <main
          className={`${styles.content} ${contentFlush ? styles.contentFlush : ''}`}
          aria-label={navAriaLabel}
        >
          {children}
        </main>
      </div>
    </div>
  )
}
