import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { PlusOutlined } from '@ant-design/icons'

import { getStudentCourses, type StudentCourse } from '@/modules/student/api'
import { CreateCourseDialog } from '@/modules/student/learning/CreateCourseDialog'
import {
  courseSourceDescriptions,
  courseSourceLabels,
  courseSourceOf,
  matchesProgressFilter,
  type CourseSource,
  type CourseSourceFilter,
  type ProgressFilter,
} from '@/modules/student/learning/courseSource'
import styles from '@/modules/student/learning/myCourses.module.css'
import homeStyles from '@/modules/student/StudentHomePage.module.css'
import { CourseCover } from '@/modules/student/studentCourseVisuals'

export function MyCoursesPage() {
  const [courses, setCourses] = useState<StudentCourse[] | null>(null)
  const [errorMessage, setErrorMessage] = useState('')
  const [createOpen, setCreateOpen] = useState(false)

  useEffect(() => {
    let cancelled = false
    getStudentCourses()
      .then((items) => {
        if (!cancelled) {
          setCourses(items)
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
  }, [])

  return (
    <div className={styles.page}>
      <div className={styles.pageHeading}>
        <div>
          <p className={styles.eyebrow}>YOUR PERSONAL LIBRARY</p>
          <h1>我的课程</h1>
          <p>创建、筛选并进入课程，继续你的学习进度。</p>
        </div>
        <button className={styles.primaryAction} type="button" onClick={() => setCreateOpen(true)}>
          <PlusOutlined /> 创建课程
        </button>
      </div>
      <CreateCourseDialog open={createOpen} onClose={() => setCreateOpen(false)} />
      {errorMessage ? (
        <div className={styles.statePanel}>
          <strong>课程加载失败</strong>
          <p>{errorMessage}</p>
        </div>
      ) : (
        <CourseLibrary courses={courses} />
      )}
    </div>
  )
}

function CourseLibrary({ courses }: { courses: StudentCourse[] | null }) {
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
      <div className={styles.skeletons} aria-label="正在加载课程">
        <span />
        <span />
        <span />
      </div>
    )
  }

  if (courses.length === 0) {
    return (
      <div className={styles.statePanel}>
        <strong>还没有课程</strong>
        <p>创建第一门课程，或等待教师发布课程到你的班级。</p>
      </div>
    )
  }

  return (
    <section aria-label="课程库">
      <div className={homeStyles.sectionHeading}>
        <div>
          <p className={styles.eyebrow}>COURSE CATALOG</p>
          <h2>全部课程</h2>
          <p className={homeStyles.sectionSub}>按来源与进度筛选，快速找到想学的课。</p>
        </div>
        <span className={homeStyles.sectionCount}>共 {filtered.length} 门课程</span>
      </div>
      <div className={styles.sourceTabs} role="tablist" aria-label="课程来源">
        {(
          [
            ['all', '全部课程'],
            ['teacher', courseSourceLabels.teacher],
            ['personal', courseSourceLabels.personal],
            ['shared', courseSourceLabels.shared],
          ] as const
        ).map(([value, label]) => (
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
            placeholder="搜索你的课程..."
            aria-label="搜索课程"
            onChange={(event) => setQuery(event.target.value)}
          />
        </label>
        <select
          value={progressFilter}
          aria-label="进度筛选"
          onChange={(event) => setProgressFilter(event.target.value as ProgressFilter)}
        >
          <option value="all">全部进度</option>
          <option value="not_started">尚未开始</option>
          <option value="in_progress">正在学习</option>
          <option value="completed">已完成</option>
        </select>
        <select value={sort} aria-label="课程排序" onChange={(event) => setSort(event.target.value as 'progress' | 'title')}>
          <option value="progress">按进度排序</option>
          <option value="title">按名称排序</option>
        </select>
      </div>
      {filtered.length === 0 ? (
        <div className={styles.statePanel}>
          <strong>没有匹配的课程</strong>
          <p>调整搜索或筛选条件后再试。</p>
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
              title={courseSourceLabels[source]}
              description={courseSourceDescriptions[source]}
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
  const source = courseSourceOf(course)
  const chapterLabel = course.chapterCount ? `${course.chapterCount} 章` : '草稿'
  const statusLabel = course.percent === 100 ? '已完成' : course.percent > 0 ? '正在学习' : '尚未开始'
  const actionLabel = course.percent > 0 ? '继续学习' : '进入课程'

  return (
    <article className={homeStyles.courseCard}>
      <Link to={`/student/courses/${course.id}`} aria-label={`打开 ${course.title}`}>
        <CourseCover title={course.title} description={course.description} />
      </Link>
      <div className={homeStyles.cardBody}>
        <div className={homeStyles.cardMeta}>
          <span className={homeStyles.pill}>{courseSourceLabels[source]}</span>
          <span className={homeStyles.cardChapter}>{chapterLabel}</span>
        </div>
        <Link className={homeStyles.cardTitle} to={`/student/courses/${course.id}`}>
          {course.title}
        </Link>
        <p className={homeStyles.cardDesc}>{course.description || '暂无课程简介'}</p>
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
