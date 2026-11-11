import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { PlusOutlined } from '@ant-design/icons'

import { useI18n } from '@/core/i18n/useI18n'
import { getStudentCourses, type StudentCourse } from '@/modules/student/api'
import { useCourseCreation } from '@/modules/student/learning/CourseCreationProvider'
import {
  courseSourceDescription,
  courseSourceLabel,
  courseSourceOf,
  matchesProgressFilter,
  progressFilterLabel,
  type CourseSource,
  type CourseSourceFilter,
  type ProgressFilter,
} from '@/modules/student/learning/courseSource'
import styles from '@/modules/student/learning/myCourses.module.css'
import homeStyles from '@/modules/student/StudentHomePage.module.css'
import { spriteForAffection } from '@/modules/student/PixelSprite'
import { CourseCover } from '@/modules/student/studentCourseVisuals'

export function MyCoursesPage() {
  const { t } = useI18n()
  const [courses, setCourses] = useState<StudentCourse[] | null>(null)
  const [errorMessage, setErrorMessage] = useState('')
  const { openDialog, revision, pendingTitle } = useCourseCreation()

  useEffect(() => {
    let cancelled = false
    getStudentCourses()
      .then((items) => {
        if (!cancelled) {
          setCourses(items)
          setErrorMessage('')
        }
      })
      .catch((error: Error) => {
        if (!cancelled) {
          setErrorMessage(error.message)
        }
      })
    return () => {
      cancelled = true
    }
  }, [revision])

  return (
    <div className={styles.page}>
      <div className={styles.pageHeading}>
        <div>
          <p className={styles.eyebrow}>{t('student.myCourses.eyebrow')}</p>
          <h1>{t('student.myCourses.title')}</h1>
          <p>{t('student.myCourses.subtitle')}</p>
        </div>
        <button className={styles.primaryAction} type="button" onClick={openDialog}>
          <PlusOutlined /> {t('student.myCourses.createCourse')}
        </button>
      </div>
      {errorMessage ? (
        <div className={styles.statePanel}>
          <strong>{t('student.myCourses.loadFailed')}</strong>
          <p>{errorMessage}</p>
        </div>
      ) : (
        <CourseLibrary courses={courses?.filter((course) => course.title !== pendingTitle || course.avatar !== null) ?? null} />
      )}
    </div>
  )
}

function CourseLibrary({ courses }: { courses: StudentCourse[] | null }) {
  const { t } = useI18n()
  const [query, setQuery] = useState('')
  const [sourceTab, setSourceTab] = useState<CourseSourceFilter>('all')
  const [progressFilter, setProgressFilter] = useState<ProgressFilter>('all')
  const [sort, setSort] = useState<'progress' | 'title'>('progress')

  const filtered = useMemo(() => {
    const keyword = query.trim().toLocaleLowerCase()
    return [...(courses ?? [])]
      .filter((course) => {
        const source = courseSourceOf(course)
        if (sourceTab !== 'all' && source !== sourceTab) {
          return false
        }
        if (!matchesProgressFilter(course, progressFilter)) {
          return false
        }
        return (
          !keyword || `${course.title} ${course.description}`.toLocaleLowerCase().includes(keyword)
        )
      })
      .sort((left, right) =>
        sort === 'title' ? left.title.localeCompare(right.title) : right.percent - left.percent,
      )
  }, [courses, progressFilter, query, sort, sourceTab])

  if (!courses) {
    return (
      <div className={styles.skeletons} aria-label={t('student.myCourses.loadingAria')}>
        <span />
        <span />
        <span />
      </div>
    )
  }

  if (courses.length === 0) {
    return (
      <div className={styles.statePanel}>
        <strong>{t('student.myCourses.emptyTitle')}</strong>
        <p>{t('student.myCourses.emptyBody')}</p>
      </div>
    )
  }

  const sourceTabs: Array<[CourseSourceFilter, string]> = [
    ['all', t('student.myCourses.tabAll')],
    ['teacher', courseSourceLabel('teacher')],
    ['personal', courseSourceLabel('personal')],
    ['shared', courseSourceLabel('shared')],
  ]

  const progressFilters: ProgressFilter[] = ['all', 'not_started', 'in_progress', 'completed']

  return (
    <section aria-label={t('student.myCourses.libraryAria')}>
      <div className={homeStyles.sectionHeading}>
        <div>
          <p className={styles.eyebrow}>{t('student.myCourses.catalogEyebrow')}</p>
          <h2>{t('student.myCourses.catalogTitle')}</h2>
          <p className={homeStyles.sectionSub}>{t('student.myCourses.catalogSub')}</p>
        </div>
        <span className={homeStyles.sectionCount}>
          {t('student.myCourses.count', { count: filtered.length })}
        </span>
      </div>
      <div className={styles.sourceTabs} role="tablist" aria-label={t('student.myCourses.sourceTabsAria')}>
        {sourceTabs.map(([value, label]) => (
          <button
            key={value}
            type="button"
            role="tab"
            aria-selected={sourceTab === value}
            className={`${styles.sourceTab} ${sourceTab === value ? styles.sourceTabActive : ''}`}
            onClick={() => setSourceTab(value)}
          >
            {label}
          </button>
        ))}
      </div>
      <div className={homeStyles.catalogToolbar}>
        <label className={homeStyles.searchField}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
            <circle cx="11" cy="11" r="7" />
            <path d="m16 16 5 5" />
          </svg>
          <input
            type="search"
            value={query}
            placeholder={t('student.myCourses.searchPlaceholder')}
            aria-label={t('student.myCourses.searchAria')}
            onChange={(event) => setQuery(event.target.value)}
          />
        </label>
        <select
          value={progressFilter}
          aria-label={t('student.myCourses.progressFilterAria')}
          onChange={(event) => setProgressFilter(event.target.value as ProgressFilter)}
        >
          {progressFilters.map((filter) => (
            <option key={filter} value={filter}>
              {progressFilterLabel(filter)}
            </option>
          ))}
        </select>
        <select
          value={sort}
          aria-label={t('student.myCourses.sortAria')}
          onChange={(event) => setSort(event.target.value as 'progress' | 'title')}
        >
          <option value="progress">{t('student.myCourses.sortByProgress')}</option>
          <option value="title">{t('student.myCourses.sortByTitle')}</option>
        </select>
      </div>
      {filtered.length === 0 ? (
        <div className={styles.statePanel}>
          <strong>{t('student.myCourses.noMatchTitle')}</strong>
          <p>{t('student.myCourses.noMatchBody')}</p>
        </div>
      ) : sourceTab === 'all' ? (
        (['teacher', 'personal', 'shared'] as CourseSource[]).map((source) => {
          const group = filtered.filter((course) => courseSourceOf(course) === source)
          if (group.length === 0) {
            return null
          }
          return (
            <CourseGroup
              key={source}
              title={courseSourceLabel(source)}
              description={courseSourceDescription(source)}
              courses={group}
            />
          )
        })
      ) : (
        <div className={styles.courseGrid}>
          {filtered.map((course) => (
            <CourseCard key={course.id} course={course} />
          ))}
        </div>
      )}
    </section>
  )
}

function CourseGroup({
  title,
  description,
  courses,
}: {
  title: string
  description: string
  courses: StudentCourse[]
}) {
  return (
    <div className={styles.groupBlock}>
      <div className={styles.groupHead}>
        <h3>{title}</h3>
        <p>{description}</p>
      </div>
      <div className={styles.courseGrid}>
        {courses.map((course) => (
          <CourseCard key={course.id} course={course} />
        ))}
      </div>
    </div>
  )
}

function CourseCard({ course }: { course: StudentCourse }) {
  const { t } = useI18n()
  const source = courseSourceOf(course)
  const chapterLabel = course.chapterCount
    ? t('student.myCourses.card.chapters', { count: course.chapterCount })
    : t('student.myCourses.card.draft')
  const statusLabel =
    course.percent === 100
      ? t('student.myCourses.card.completed')
      : course.percent > 0
        ? t('student.myCourses.card.inProgress')
        : t('student.myCourses.card.notStarted')
  const actionLabel =
    course.percent > 0 ? t('student.myCourses.card.continue') : t('student.myCourses.card.enter')

  return (
    <article className={homeStyles.courseCard}>
      <Link to={`/student/courses/${course.id}`} aria-label={t('student.myCourses.card.openAria', { title: course.title })}>
        <CourseCover
          title={course.title}
          description={course.description}
          courseId={course.id}
          sprite={course.avatar ? spriteForAffection(course.avatar, course.affection) : null}
          affection={course.affection}
        />
      </Link>
      <div className={homeStyles.cardBody}>
        <div className={homeStyles.cardMeta}>
          <span className={homeStyles.pill}>{courseSourceLabel(source)}</span>
          <span className={homeStyles.cardChapter}>{chapterLabel}</span>
        </div>
        <Link className={homeStyles.cardTitle} to={`/student/courses/${course.id}`}>
          {course.title}
        </Link>
        <p className={homeStyles.cardDesc}>{course.description || t('student.myCourses.card.noDescription')}</p>
        <div className={homeStyles.cardBottom}>
          <div className={homeStyles.cardMeter}>
            <div className={homeStyles.meter} aria-hidden="true">
              <span className={homeStyles.meterFill} style={{ width: `${course.percent}%` }} />
            </div>
            <b className={homeStyles.percent}>{course.percent}%</b>
          </div>
          <div className={homeStyles.cardAction}>
            <span>{statusLabel}</span>
            <Link className={homeStyles.cardActionLink} to={`/student/courses/${course.id}`}>
              {actionLabel}
            </Link>
          </div>
        </div>
      </div>
    </article>
  )
}
