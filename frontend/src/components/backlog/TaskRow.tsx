import { useState } from 'react'
import { useMutation } from '@apollo/client/react'
import { COMPLETE_TASK, UPDATE_TASK, DELETE_TASK, GET_TASKS } from '../../graphql/backlog'
import { CATEGORY_COLORS } from '../../constants'

interface Task {
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
  parentId?: string | null
}

interface Props {
  task: Task
  isSubtask?: boolean
}

function formatMinutes(mins: number): string {
  if (mins < 60) return `${mins}min`
  const h = Math.floor(mins / 60)
  const m = mins % 60
  return m > 0 ? `${h}hr ${m}min` : `${h}hr`
}

function formatDeadline(task: Task): { text: string; overdue: boolean } | null {
  if (task.deadlineType === 'HARD' && task.deadlineDate) {
    const d = new Date(task.deadlineDate + 'T00:00:00')
    const now = new Date()
    now.setHours(0, 0, 0, 0)
    const overdue = d < now
    const formatted = d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
    return { text: overdue ? `OVERDUE — due ${formatted}` : `due ${formatted}`, overdue }
  }
  if (task.deadlineType === 'HORIZON' && task.deadlineDays) {
    return { text: `within ${task.deadlineDays} days`, overdue: false }
  }
  return null
}

const PRIORITY_COLORS: Record<string, string> = {
  HIGH: 'text-red-400',
  MEDIUM: 'text-amber-400',
  LOW: 'text-text-secondary',
}

export default function TaskRow({ task, isSubtask }: Props) {
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState(task.title)

  const [completeTask] = useMutation(COMPLETE_TASK, {
    refetchQueries: [{ query: GET_TASKS, variables: { includeCompleted: true } }],
  })
  const [updateTask] = useMutation(UPDATE_TASK, {
    refetchQueries: [{ query: GET_TASKS, variables: { includeCompleted: true } }],
  })
  const [deleteTask] = useMutation(DELETE_TASK, {
    refetchQueries: [{ query: GET_TASKS, variables: { includeCompleted: true } }],
  })

  const handleComplete = async () => {
    if (task.isCompleted) {
      await updateTask({ variables: { id: task.id, input: { isCompleted: false } } })
    } else {
      await completeTask({ variables: { id: task.id } })
    }
  }

  const handleTitleSave = async () => {
    setEditingTitle(false)
    if (titleDraft.trim() && titleDraft !== task.title) {
      await updateTask({ variables: { id: task.id, input: { title: titleDraft.trim() } } })
    } else {
      setTitleDraft(task.title)
    }
  }

  const handleDelete = async () => {
    const hasSubtasks = !isSubtask && !task.parentId
    const msg = hasSubtasks ? 'Delete this task and all subtasks?' : 'Delete this task?'
    if (window.confirm(msg)) {
      await deleteTask({ variables: { id: task.id } })
    }
  }

  const color = CATEGORY_COLORS[task.category] || '#6b7280'
  const deadline = formatDeadline(task)

  return (
    <div
      className={`flex items-center gap-3 p-3 rounded-lg bg-bg-surface mb-2 group ${
        isSubtask ? 'ml-6 text-sm' : ''
      }`}
      style={{ borderLeft: `4px solid ${color}` }}
    >
      {/* Checkbox */}
      <button
        onClick={handleComplete}
        className={`w-5 h-5 rounded border flex-shrink-0 flex items-center justify-center transition-colors ${
          task.isCompleted
            ? 'bg-accent border-accent text-black'
            : 'border-border-default hover:border-accent'
        }`}
      >
        {task.isCompleted && <span className="text-xs">✓</span>}
      </button>

      {/* Title */}
      <div className="flex-1 min-w-0">
        {editingTitle ? (
          <input
            value={titleDraft}
            onChange={(e) => setTitleDraft(e.target.value)}
            onBlur={handleTitleSave}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleTitleSave()
              if (e.key === 'Escape') { setEditingTitle(false); setTitleDraft(task.title) }
            }}
            autoFocus
            className="w-full bg-transparent border-b border-accent text-text-primary outline-none"
          />
        ) : (
          <span
            onClick={() => setEditingTitle(true)}
            className={`cursor-pointer ${task.isCompleted ? 'text-text-secondary line-through' : 'text-text-primary'}`}
          >
            {task.title}
          </span>
        )}
      </div>

      {/* Deferred indicator */}
      {task.timesDeferred >= 2 && (
        <span className="text-amber-400 text-xs flex-shrink-0">deferred {task.timesDeferred}x</span>
      )}

      {/* Priority */}
      <span className={`text-xs flex-shrink-0 ${PRIORITY_COLORS[task.priority] || ''}`}>
        {task.priority}
      </span>

      {/* Progress (subtask/standalone) */}
      {task.estimatedMinutes != null && (
        <span className="text-text-secondary text-xs flex-shrink-0">
          {formatMinutes(task.actualMinutes)} / {formatMinutes(task.estimatedMinutes)}
        </span>
      )}

      {/* Deadline */}
      {deadline && (
        <span className={`text-xs flex-shrink-0 ${deadline.overdue ? 'text-red-400' : 'text-text-secondary'}`}>
          {deadline.text}
        </span>
      )}

      {/* Delete */}
      <button
        onClick={handleDelete}
        className="text-text-secondary hover:text-red-400 opacity-0 group-hover:opacity-100 transition-all text-sm flex-shrink-0"
        title="Delete"
      >
        🗑
      </button>
    </div>
  )
}
