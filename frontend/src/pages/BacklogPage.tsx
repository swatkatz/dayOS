import { useState, useMemo } from 'react'
import { useQuery } from '@apollo/client/react'
import { GET_TASKS } from '../graphql/backlog'
import TaskGroup from '../components/backlog/TaskGroup'
import TaskRow from '../components/backlog/TaskRow'
import QuickAddForm from '../components/backlog/QuickAddForm'
import TaskFilters from '../components/backlog/TaskFilters'
import ScopeChat from '../components/backlog/ScopeChat'
import CompletedSection from '../components/backlog/CompletedSection'

interface Task {
  id: string
  title: string
  category: string
  priority: string
  parentId: string | null
  estimatedMinutes: number | null
  actualMinutes: number
  deadlineType: string | null
  deadlineDate: string | null
  deadlineDays: number | null
  notes: string | null
  isRoutine: boolean
  timesDeferred: number
  isCompleted: boolean
  completedAt: string | null
  createdAt: string
  subtasks: Array<{
    id: string
    title: string
    category: string
    priority: string
    estimatedMinutes: number | null
    actualMinutes: number
    deadlineType: string | null
    deadlineDate: string | null
    deadlineDays: number | null
    notes: string | null
    isCompleted: boolean
    completedAt: string | null
    timesDeferred: number
  }>
}

export default function BacklogPage() {
  const [showQuickAdd, setShowQuickAdd] = useState(false)
  const [showScopeChat, setShowScopeChat] = useState(false)
  const [categoryFilter, setCategoryFilter] = useState('All')
  const [priorityFilter, setPriorityFilter] = useState('All')

  const { data, loading, error } = useQuery<{ tasks: Task[] }>(GET_TASKS, {
    variables: { includeCompleted: true },
  })

  const { parentTasks, standaloneTasks, completedTasks } = useMemo(() => {
    const tasks: Task[] = data?.tasks ?? []

    const matchesFilter = (t: { category: string; priority: string }) => {
      if (categoryFilter !== 'All' && t.category !== categoryFilter) return false
      if (priorityFilter !== 'All' && t.priority !== priorityFilter) return false
      return true
    }

    const parents: Task[] = []
    const standalone: Task[] = []
    const completed: Task[] = []

    for (const task of tasks) {
      if (task.parentId) continue // Skip subtasks at top level

      const isParent = task.subtasks.length > 0
      const allSubtasksCompleted = isParent && task.subtasks.every((s) => s.isCompleted)

      if (task.isCompleted || allSubtasksCompleted) {
        if (matchesFilter(task)) completed.push(task)
      } else if (isParent) {
        if (matchesFilter(task)) parents.push(task)
      } else {
        if (matchesFilter(task)) standalone.push(task)
      }
    }

    return { parentTasks: parents, standaloneTasks: standalone, completedTasks: completed }
  }, [data, categoryFilter, priorityFilter])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64 text-text-secondary">Loading...</div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-64 gap-3">
        <p className="text-red-400">Failed to load tasks</p>
        <button
          onClick={() => window.location.reload()}
          className="px-4 py-2 bg-accent text-black rounded font-medium hover:bg-accent-hover"
        >
          Retry
        </button>
      </div>
    )
  }

  return (
    <div className="max-w-3xl mx-auto">
      {/* Header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between mb-6">
        <h1 className="text-xl md:text-2xl font-semibold text-text-primary">Task Backlog</h1>
        <div className="flex gap-2">
          <button
            onClick={() => setShowQuickAdd(true)}
            className="flex-1 sm:flex-none px-4 py-2.5 bg-accent text-black rounded-xl font-medium hover:bg-accent-hover active:scale-[0.98] transition-all"
          >
            + Quick Add
          </button>
          <button
            onClick={() => setShowScopeChat(true)}
            className="flex-1 sm:flex-none px-4 py-2.5 border border-border-default text-text-primary rounded-xl hover:border-accent active:scale-[0.98] transition-all"
          >
            Scope with AI
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="mb-6">
        <TaskFilters
          category={categoryFilter}
          priority={priorityFilter}
          onCategoryChange={setCategoryFilter}
          onPriorityChange={setPriorityFilter}
        />
      </div>

      {/* Quick Add Form */}
      {showQuickAdd && <QuickAddForm onClose={() => setShowQuickAdd(false)} />}

      {/* Parent Tasks */}
      {parentTasks.length > 0 && (
        <div className="mb-6">
          <h2 className="text-sm font-medium text-text-secondary uppercase tracking-wide border-b border-border-default pb-2 mb-3">
            Parent Tasks
          </h2>
          {parentTasks.map((task) => (
            <TaskGroup key={task.id} task={task} />
          ))}
        </div>
      )}

      {/* Standalone Tasks */}
      {standaloneTasks.length > 0 && (
        <div className="mb-6">
          <h2 className="text-sm font-medium text-text-secondary uppercase tracking-wide border-b border-border-default pb-2 mb-3">
            Standalone Tasks
          </h2>
          {standaloneTasks.map((task) => (
            <TaskRow key={task.id} task={task} />
          ))}
        </div>
      )}

      {/* Empty state */}
      {parentTasks.length === 0 && standaloneTasks.length === 0 && completedTasks.length === 0 && (
        <div className="text-center text-text-secondary py-16">
          No tasks yet. Add one with Quick Add or Scope with AI.
        </div>
      )}

      {/* Completed */}
      <CompletedSection tasks={completedTasks} />

      {/* Scope Chat Panel */}
      {showScopeChat && <ScopeChat onClose={() => setShowScopeChat(false)} />}
    </div>
  )
}
