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
  readOnly?: boolean
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

const CATEGORIES = ['JOB', 'INTERVIEW', 'PROJECT', 'MEAL', 'BABY', 'EXERCISE', 'ADMIN']
const PRIORITIES = ['HIGH', 'MEDIUM', 'LOW']

export default function TaskRow({ task, isSubtask, readOnly }: Props) {
  const locked = task.isCompleted || !!readOnly
  const [expanded, setExpanded] = useState(false)
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState(task.title)
  const [editingNotes, setEditingNotes] = useState(false)
  const [notesDraft, setNotesDraft] = useState(task.notes ?? '')
  const [editingPriority, setEditingPriority] = useState(false)
  const [editingEst, setEditingEst] = useState(false)
  const [estDraft, setEstDraft] = useState(String(task.estimatedMinutes ?? ''))

  const refetchOpts = { refetchQueries: [{ query: GET_TASKS, variables: { includeCompleted: true } }] }
  const [completeTask] = useMutation(COMPLETE_TASK, refetchOpts)
  const [updateTask] = useMutation(UPDATE_TASK, refetchOpts)
  const [deleteTask] = useMutation(DELETE_TASK, refetchOpts)

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

  const handleNotesSave = async () => {
    setEditingNotes(false)
    const trimmed = notesDraft.trim()
    if (trimmed !== (task.notes ?? '')) {
      await updateTask({ variables: { id: task.id, input: { notes: trimmed || null } } })
    }
  }

  const handleDelete = async () => {
    const hasSubtasks = !isSubtask && !task.parentId
    const msg = hasSubtasks ? 'Delete this task and all subtasks?' : 'Delete this task?'
    if (window.confirm(msg)) {
      await deleteTask({ variables: { id: task.id } })
    }
  }

  const handleFieldUpdate = async (input: Record<string, unknown>) => {
    await updateTask({ variables: { id: task.id, input } })
  }

  const color = CATEGORY_COLORS[task.category] || '#6b7280'
  const deadline = formatDeadline(task)

  return (
    <div className={`mb-2 ${isSubtask ? 'ml-6' : ''}`}>
      {/* Main row */}
      <div
        className={`flex items-center gap-2 md:gap-3 p-3 rounded-xl bg-bg-surface group ${isSubtask ? 'text-sm' : ''}`}
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

        {/* Title + Notes */}
        <div className="flex-1 min-w-0">
          {!locked && editingTitle ? (
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
              onClick={!locked ? () => setEditingTitle(true) : undefined}
              className={`${!locked ? 'cursor-pointer' : ''} ${task.isCompleted ? 'text-text-secondary line-through' : 'text-text-primary'}`}
            >
              {task.title}
            </span>
          )}
          {!locked && editingNotes ? (
            <textarea
              value={notesDraft}
              onChange={(e) => setNotesDraft(e.target.value)}
              onBlur={handleNotesSave}
              onKeyDown={(e) => {
                if (e.key === 'Escape') { setEditingNotes(false); setNotesDraft(task.notes ?? '') }
              }}
              autoFocus
              rows={2}
              className="w-full mt-1 bg-transparent border border-accent rounded px-1 py-0.5 text-text-secondary text-xs outline-none resize-y"
              placeholder="Add notes..."
            />
          ) : task.notes ? (
            <p
              onClick={!locked ? () => setEditingNotes(true) : undefined}
              className={`text-text-secondary text-xs mt-1 whitespace-pre-wrap line-clamp-2 ${!locked ? 'cursor-pointer hover:text-text-primary' : ''} transition-colors`}
            >
              {task.notes}
            </p>
          ) : !locked ? (
            <p
              onClick={() => setEditingNotes(true)}
              className="text-text-secondary text-xs mt-1 cursor-pointer hover:text-text-primary transition-colors"
            >
              <span className="opacity-0 group-hover:opacity-100">+ Add notes</span>
            </p>
          ) : null}
        </div>

        {/* Deferred indicator */}
        {task.timesDeferred >= 2 && (
          <span className="text-amber-400 text-xs flex-shrink-0">deferred {task.timesDeferred}x</span>
        )}

        {/* Priority */}
        {!locked && editingPriority ? (
          <select
            value={task.priority}
            onChange={(e) => { handleFieldUpdate({ priority: e.target.value }); setEditingPriority(false) }}
            onBlur={() => setEditingPriority(false)}
            autoFocus
            className="text-xs flex-shrink-0 bg-bg-primary border border-accent rounded px-1 py-0.5 text-text-primary outline-none"
          >
            {PRIORITIES.map((p) => <option key={p} value={p}>{p}</option>)}
          </select>
        ) : (
          <span
            onClick={!locked ? () => setEditingPriority(true) : undefined}
            className={`text-xs flex-shrink-0 ${!locked ? 'cursor-pointer hover:opacity-75' : ''} transition-opacity ${PRIORITY_COLORS[task.priority] || ''}`}
          >
            {task.priority}
          </span>
        )}

        {/* Progress (subtask/standalone) */}
        {!locked && editingEst ? (
          <div className="flex items-center gap-1 flex-shrink-0">
            <input
              type="number"
              value={estDraft}
              onChange={(e) => setEstDraft(e.target.value)}
              onBlur={() => {
                setEditingEst(false)
                const v = parseInt(estDraft)
                if (!isNaN(v) && v > 0 && v !== task.estimatedMinutes) handleFieldUpdate({ estimatedMinutes: v })
              }}
              onKeyDown={(e) => {
                if (e.key === 'Enter') (e.target as HTMLInputElement).blur()
                if (e.key === 'Escape') { setEditingEst(false); setEstDraft(String(task.estimatedMinutes ?? '')) }
              }}
              autoFocus
              className="w-14 bg-bg-primary border border-accent rounded px-1 py-0.5 text-xs text-text-primary outline-none"
            />
            <span className="text-text-secondary text-xs">min</span>
          </div>
        ) : (
          <span
            onClick={!locked ? () => { setEstDraft(String(task.estimatedMinutes ?? '')); setEditingEst(true) } : undefined}
            className={`text-text-secondary text-xs flex-shrink-0 ${!locked ? 'cursor-pointer hover:text-text-primary' : ''} transition-colors`}
          >
            {task.estimatedMinutes != null
              ? `${formatMinutes(task.actualMinutes)} / ${formatMinutes(task.estimatedMinutes)}`
              : !locked ? <span className="opacity-0 group-hover:opacity-100">+ est.</span> : null}
          </span>
        )}

        {/* Deadline */}
        {deadline && (
          <span className={`text-xs flex-shrink-0 ${deadline.overdue ? 'text-red-400' : 'text-text-secondary'}`}>
            {deadline.text}
          </span>
        )}

        {!locked && (
          <>
            {/* Expand */}
            <button
              onClick={() => setExpanded(!expanded)}
              className="text-text-secondary hover:text-text-primary opacity-0 group-hover:opacity-100 transition-all text-sm flex-shrink-0"
              title="Edit"
            >
              {expanded ? '▲' : '▼'}
            </button>

            {/* Delete */}
            <button
              onClick={handleDelete}
              className="text-text-secondary hover:text-red-400 opacity-0 group-hover:opacity-100 transition-all text-sm flex-shrink-0"
              title="Delete"
            >
              🗑
            </button>
          </>
        )}
      </div>

      {/* Expanded edit panel */}
      {expanded && (
        <div className="bg-bg-surface rounded-b-xl px-3 md:px-4 py-3 border-t border-border-default space-y-3 text-sm" style={{ borderLeft: `4px solid ${color}` }}>
          <div className="flex gap-4 flex-wrap">
            {/* Category */}
            <label className="flex flex-col gap-1">
              <span className="text-text-secondary text-xs">Category</span>
              <select
                value={task.category}
                onChange={(e) => handleFieldUpdate({ category: e.target.value })}
                className="bg-bg-primary border border-border-default rounded px-2 py-1 text-text-primary"
              >
                {CATEGORIES.map((c) => <option key={c} value={c}>{c}</option>)}
              </select>
            </label>

            {/* Priority */}
            <label className="flex flex-col gap-1">
              <span className="text-text-secondary text-xs">Priority</span>
              <select
                value={task.priority}
                onChange={(e) => handleFieldUpdate({ priority: e.target.value })}
                className="bg-bg-primary border border-border-default rounded px-2 py-1 text-text-primary"
              >
                {PRIORITIES.map((p) => <option key={p} value={p}>{p}</option>)}
              </select>
            </label>

            {/* Estimated minutes */}
            <label className="flex flex-col gap-1">
              <span className="text-text-secondary text-xs">Est. minutes</span>
              <input
                type="number"
                defaultValue={task.estimatedMinutes ?? ''}
                onBlur={(e) => {
                  const v = parseInt(e.target.value)
                  if (!isNaN(v) && v > 0) handleFieldUpdate({ estimatedMinutes: v })
                }}
                onKeyDown={(e) => { if (e.key === 'Enter') (e.target as HTMLInputElement).blur() }}
                className="bg-bg-primary border border-border-default rounded px-2 py-1 text-text-primary w-20"
              />
            </label>
          </div>

          {/* Deadline row */}
          <div className="flex gap-4 flex-wrap items-end">
            <label className="flex flex-col gap-1">
              <span className="text-text-secondary text-xs">Deadline type</span>
              <select
                value={task.deadlineType ?? ''}
                onChange={(e) => {
                  const val = e.target.value
                  if (val === '') {
                    handleFieldUpdate({ deadlineType: null, deadlineDate: null, deadlineDays: null })
                  } else {
                    handleFieldUpdate({ deadlineType: val })
                  }
                }}
                className="bg-bg-primary border border-border-default rounded px-2 py-1 text-text-primary"
              >
                <option value="">None</option>
                <option value="HARD">Hard date</option>
                <option value="HORIZON">Horizon</option>
              </select>
            </label>

            {task.deadlineType === 'HARD' && (
              <label className="flex flex-col gap-1">
                <span className="text-text-secondary text-xs">Date</span>
                <input
                  type="date"
                  defaultValue={task.deadlineDate ?? ''}
                  onBlur={(e) => {
                    if (e.target.value) handleFieldUpdate({ deadlineDate: e.target.value })
                  }}
                  className="bg-bg-primary border border-border-default rounded px-2 py-1 text-text-primary"
                />
              </label>
            )}

            {task.deadlineType === 'HORIZON' && (
              <label className="flex flex-col gap-1">
                <span className="text-text-secondary text-xs">Within N days</span>
                <input
                  type="number"
                  defaultValue={task.deadlineDays ?? ''}
                  onBlur={(e) => {
                    const v = parseInt(e.target.value)
                    if (!isNaN(v) && v > 0) handleFieldUpdate({ deadlineDays: v })
                  }}
                  onKeyDown={(e) => { if (e.key === 'Enter') (e.target as HTMLInputElement).blur() }}
                  className="bg-bg-primary border border-border-default rounded px-2 py-1 text-text-primary w-20"
                />
              </label>
            )}
          </div>

          {/* Notes */}
          <label className="flex flex-col gap-1">
            <span className="text-text-secondary text-xs">Notes</span>
            <textarea
              defaultValue={task.notes ?? ''}
              onBlur={(e) => {
                const val = e.target.value.trim()
                if (val !== (task.notes ?? '')) handleFieldUpdate({ notes: val || null })
              }}
              rows={2}
              className="bg-bg-primary border border-border-default rounded px-2 py-1 text-text-primary resize-y"
              placeholder="Add notes..."
            />
          </label>
        </div>
      )}
    </div>
  )
}
