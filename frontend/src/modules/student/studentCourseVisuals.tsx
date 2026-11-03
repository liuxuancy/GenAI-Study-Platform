import type { ReactNode } from 'react'

import styles from '@/modules/student/StudentHomePage.module.css'

export type CourseCoverKey =
  | 'notes'
  | 'reading'
  | 'git'
  | 'db'
  | 'algo'
  | 'line'
  | 'java'
  | 'sort'
  | 'introdb'

const coverMeta: Record<CourseCoverKey, { label: string; className: string }> = {
  notes: { label: 'STUDY JOURNAL', className: 'coverNotes' },
  reading: { label: 'READING SKILLS', className: 'coverReading' },
  git: { label: 'DEVELOPER TOOLS', className: 'coverGit' },
  db: { label: 'COMPUTER SCIENCE', className: 'coverDb' },
  algo: { label: 'ALGORITHMS', className: 'coverAlgo' },
  line: { label: 'STUDY NOTES', className: 'coverLine' },
  java: { label: 'PROGRAMMING', className: 'coverJava' },
  sort: { label: 'ALGORITHMS', className: 'coverSort' },
  introdb: { label: 'DATABASE BASICS', className: 'coverIntrodb' },
}

const coverSvgs: Record<CourseCoverKey, string> = {
  notes: `<svg viewBox="0 0 360 160" fill="none"><rect x="111" y="30" width="136" height="109" rx="8" fill="#fff8e7" stroke="#bf9b63" stroke-width="2" transform="rotate(-9 111 30)"/><rect x="143" y="31" width="136" height="109" rx="8" fill="#fff3d8" stroke="#bb945b" stroke-width="2" transform="rotate(6 143 31)"/><path d="M163 65h84M163 83h70M163 101h78" stroke="#bfa070" stroke-width="3" stroke-linecap="round"/><path d="M224 25l19 22-37 34-19-21z" fill="#d99165"/><circle cx="94" cy="118" r="25" fill="#ecc782" opacity=".6"/></svg>`,
  reading: `<svg viewBox="0 0 360 160" fill="none"><circle cx="260" cy="45" r="54" fill="#c7e3d0"/><path d="M91 39q56-17 89 7v82q-34-24-89-7z" fill="#fffef4" stroke="#5b8d68" stroke-width="2"/><path d="M180 46q44-24 94-8v83q-50-16-94 7z" fill="#f8f9ec" stroke="#5b8d68" stroke-width="2"/><path d="M180 46v82M107 61h52M107 76h42M199 61h53M199 76h39M199 90h46" stroke="#92b69b" stroke-width="2"/><path d="M236 30v51l-9-7-9 7V30" fill="#f2a47d"/><circle cx="82" cy="103" r="8" fill="#a9d2ae"/></svg>`,
  git: `<svg viewBox="0 0 360 160" fill="none"><circle cx="263" cy="80" r="77" fill="#f1c4b0"/><rect x="92" y="20" width="175" height="119" rx="12" fill="#443d47" transform="rotate(-8 92 20)"/><rect x="99" y="27" width="162" height="25" rx="8" fill="#625663" transform="rotate(-8 99 27)"/><circle cx="115" cy="38" r="4" fill="#fd8e70"/><circle cx="128" cy="36" r="4" fill="#fad48a"/><path d="M123 77l13 10-13 10M148 99h41" stroke="#a5d5c4" stroke-width="5" stroke-linecap="round" stroke-linejoin="round"/><path d="M210 70v41M210 70l20 15-20 26" stroke="#f8ad89" stroke-width="4"/><circle cx="210" cy="69" r="7" fill="#ffc39c"/><circle cx="231" cy="85" r="7" fill="#ffc39c"/><circle cx="210" cy="112" r="7" fill="#ffc39c"/></svg>`,
  db: `<svg viewBox="0 0 360 160" fill="none"><circle cx="270" cy="76" r="72" fill="#c5deeb"/><ellipse cx="173" cy="45" rx="77" ry="23" fill="#eff8fc" stroke="#4c839d" stroke-width="3"/><path d="M96 45v67c0 13 34 23 77 23s77-10 77-23V45" fill="#c4e4f2" stroke="#4c839d" stroke-width="3"/><path d="M96 66c0 13 34 23 77 23s77-10 77-23M96 90c0 13 34 23 77 23s77-10 77-23" stroke="#4c839d" stroke-width="3"/><ellipse cx="173" cy="45" rx="77" ry="23" fill="#eff8fc" stroke="#4c839d" stroke-width="3"/><circle cx="119" cy="104" r="5" fill="#e4a16d"/><circle cx="119" cy="79" r="5" fill="#e4a16d"/></svg>`,
  algo: `<svg viewBox="0 0 360 160" fill="none"><circle cx="250" cy="75" r="70" fill="#d9c5f3"/><path d="M104 37l75 42 77-45M179 79v42M104 37v68l75 43 77-43V34" stroke="#8f6abb" stroke-width="3"/><rect x="91" y="22" width="28" height="28" rx="8" fill="#7453a9"/><rect x="166" y="65" width="28" height="28" rx="8" fill="#9272c1"/><rect x="243" y="21" width="28" height="28" rx="8" fill="#7453a9"/><rect x="165" y="123" width="28" height="28" rx="8" fill="#ad8ed5"/><circle cx="104" cy="106" r="9" fill="#c2a4df"/><circle cx="256" cy="105" r="9" fill="#c2a4df"/></svg>`,
  line: `<svg viewBox="0 0 360 160" fill="none"><circle cx="255" cy="78" r="70" fill="#edb8ca"/><rect x="92" y="31" width="189" height="98" rx="13" fill="#fff9fb" stroke="#b77e95" stroke-width="2"/><path d="M112 54h98M112 69h144M112 84h82" stroke="#d7afbf" stroke-width="3" stroke-linecap="round"/><path d="M138 112c18-41 42 22 61-20s31 20 52-4" stroke="#c77594" stroke-width="4" stroke-linecap="round"/><circle cx="138" cy="112" r="6" fill="#bc698b"/><circle cx="251" cy="88" r="6" fill="#bc698b"/></svg>`,
  java: `<svg viewBox="0 0 360 160" fill="none"><circle cx="260" cy="70" r="70" fill="#c4d2f5"/><path d="M135 85h109v28c0 22-29 30-56 30s-53-9-53-30z" fill="#fff" stroke="#6077b0" stroke-width="3"/><path d="M244 88h16c27 0 16 39-17 35" stroke="#6077b0" stroke-width="7" stroke-linecap="round"/><path d="M168 74c-29-25 29-32 0-51M200 75c-28-25 29-35 3-58" stroke="#eaa070" stroke-width="4" stroke-linecap="round"/><ellipse cx="190" cy="145" rx="84" ry="7" fill="#9cb1e5"/></svg>`,
  sort: `<svg viewBox="0 0 360 160" fill="none"><circle cx="249" cy="74" r="70" fill="#c2e9d8"/><rect x="105" y="101" width="28" height="32" rx="5" fill="#9dd3b7"/><rect x="143" y="75" width="28" height="58" rx="5" fill="#73bca0"/><rect x="181" y="40" width="28" height="93" rx="5" fill="#439b7d"/><rect x="219" y="58" width="28" height="75" rx="5" fill="#5eaf91"/><path d="M112 54l33-21 27 11 47-24" stroke="#398a71" stroke-width="4" stroke-linecap="round" stroke-linejoin="round"/><path d="m211 21 10-3-2 12" stroke="#398a71" stroke-width="4" stroke-linecap="round"/></svg>`,
  introdb: `<svg viewBox="0 0 360 160" fill="none"><circle cx="259" cy="81" r="72" fill="#f1d0ac"/><rect x="106" y="38" width="144" height="87" rx="11" fill="#fff9ef" stroke="#b88e60" stroke-width="3"/><ellipse cx="178" cy="50" rx="60" ry="16" fill="#e7b883" stroke="#b88e60" stroke-width="3"/><path d="M118 50v42c0 9 27 17 60 17s60-8 60-17V50M118 72c0 9 27 17 60 17s60-8 60-17" stroke="#b88e60" stroke-width="3"/><circle cx="271" cy="49" r="12" fill="#e7a775"/><path d="M271 61v40m0 0 19 20m-19-20-22 18" stroke="#c58f67" stroke-width="4"/></svg>`,
}

export function resolveCourseCoverKey(title: string, description = ''): CourseCoverKey {
  const text = `${title} ${description}`.toLowerCase()
  const name = title.trim().toLowerCase()

  if (/git/.test(text)) return 'git'
  if (name === 'database') return 'introdb'
  if (/sort/.test(text)) return 'sort'
  if (name === 'java' || /\bjava\b/.test(text)) return 'java'
  if (/database|sql systems|database systems/.test(text)) return 'db'
  if (/algorithm|data structure/.test(text)) return 'algo'
  if (/lecture|how a lecture/.test(text)) return 'reading'
  if (/shared line|asking on a line/.test(text)) return 'line'
  if (/notes/.test(text)) return 'notes'
  return 'notes'
}

const dashboardAccent: Record<CourseCoverKey, { color: string; symbol: string }> = {
  notes: { color: '#f6dfe9', symbol: '✦' },
  reading: { color: '#dcebf3', symbol: '📖' },
  git: { color: '#f8e1d6', symbol: '⌘' },
  db: { color: '#dcebf3', symbol: '▤' },
  algo: { color: '#e9e0fa', symbol: '◇' },
  line: { color: '#f6dfe9', symbol: '✦' },
  java: { color: '#c4d2f5', symbol: '☕' },
  sort: { color: '#c2e9d8', symbol: '▥' },
  introdb: { color: '#f1d0ac', symbol: '▤' },
}

export function courseDashboardAccent(title: string, description = '') {
  const key = resolveCourseCoverKey(title, description)
  return dashboardAccent[key]
}

export function courseTopicLabel(title: string, description = '') {
  const key = resolveCourseCoverKey(title, description)
  const topics: Record<CourseCoverKey, string> = {
    notes: '个人学习',
    reading: '阅读与理解',
    git: '版本控制',
    db: '数据库系统',
    algo: '算法与结构',
    line: '课堂笔记',
    java: 'Java 开发',
    sort: '排序算法',
    introdb: '数据入门',
  }
  return topics[key]
}

export function CourseCover({
  title,
  description,
}: {
  title: string
  description?: string
}) {
  const key = resolveCourseCoverKey(title, description)
  const meta = coverMeta[key]
  const toneClass = styles[meta.className as keyof typeof styles]

  return (
    <div className={`${styles.cover} ${toneClass ?? ''}`}>
      <div className={styles.coverTop}>{meta.label}</div>
      <div
        className={styles.coverArt}
        aria-hidden="true"
        dangerouslySetInnerHTML={{ __html: coverSvgs[key] }}
      />
      <div className={styles.coverBottom}>{courseTopicLabel(title, description)}</div>
    </div>
  )
}

export function FeaturedHeroArt({ title, description }: { title: string; description?: string }) {
  const key = resolveCourseCoverKey(title, description)

  if (key === 'git') {
    return (
      <div className={styles.heroArt} aria-hidden="true">
        <div className={styles.terminal}>
          <div className={styles.terminalTop}>
            <i />
            <i />
            <i />
          </div>
          <div className={styles.termCode}>
            <div>
              <span className={styles.termGreen}>❯</span> git status
            </div>
            <div>
              On branch <span className={styles.termOrange}>main</span>
            </div>
            <div>
              <span className={styles.termGreen}>❯</span> git add .
            </div>
            <div>
              <span className={styles.termGreen}>❯</span> git commit -m{' '}
              <span className={styles.termOrange}>&quot;learn&quot;</span>
            </div>
          </div>
        </div>
        <div className={styles.gitBadge}>git</div>
      </div>
    )
  }

  return (
    <div className={styles.heroArt} aria-hidden="true">
      <div
        className={styles.heroIllustration}
        dangerouslySetInnerHTML={{ __html: coverSvgs[key] }}
      />
      <div className={styles.heroBadge}>{courseTopicLabel(title, description)}</div>
    </div>
  )
}

export function StatIcon({ children }: { children: ReactNode }) {
  return <div className={styles.statIcon}>{children}</div>
}
