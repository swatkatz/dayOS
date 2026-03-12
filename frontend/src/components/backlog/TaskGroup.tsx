import { CATEGORY_COLORS } from '../../constants'
import TaskRow from './TaskRow'

interface Subtask {
  id: string
  title: string
  category: string
  priority: string
  estimatedMinutes: number | null
  actualMinutes: number
  notes: string | null
  isCompleted: boolean
  completedAt: string | null
  timesDeferred: number
}

interface ParentTask {
  id: string
  title: string
  category: string
  priority: string
  deadlineType: string | null
  deadlineDate: string | null
  deadlineDays: number | null
  subtasks: Subtask[]
}

interface Props {
  task: ParentTask
}

function formatDeadlineShort(task: ParentTask): string | null {
  if (task.deadlineType === 'HARD' && task.deadlineDate) {
    const d = new Date(task.deadlineDate + 'T00:00:00')
    return `due ${d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}`
  }
  if (task.deadlineType === 'HORIZON' && task.deadlineDays) {
    return `within ${task.deadlineDays}d`
  }
  return null
}

export default function TaskGroup({ task }: Props) {
  const color = CATEGORY_COLORS[task.category] || '#6b7280'
  const completedCount = task.subtasks.filter((s) => s.isCompleted).length
  const totalCount = task.subtasks.length
  const progress = totalCount > 0 ? (completedCount / totalCount) * 100 : 0
  const deadline = formatDeadlineShort(task)

  return (
    <div className="mb-4">
      <div className="flex items-center gap-3 mb-2">
        <span className="text-text-primary font-medium">◈ {task.title}</span>
        <span className="text-xs px-2 py-0.5 rounded" style={{ color, borderColor: color, border: '1px solid' }}>
          {task.category.toLowerCase()}
        </span>
        {deadline && <span className="text-text-secondary text-xs">{deadline}</span>}
      </div>

      {/* Progress bar */}
      <div className="flex items-center gap-2 mb-2 ml-4">
        <div className="flex-1 h-1.5 bg-bg-surface-hover rounded-full max-w-48">
          <div
            className="h-full rounded-full transition-all"
            style={{ width: `${progress}%`, backgroundColor: color }}
          />
        </div>
        <span className="text-text-secondary text-xs">{completedCount}/{totalCount} subtasks</span>
      </div>

      {/* Subtasks */}
      {task.subtasks.map((subtask) => (
        <TaskRow
          key={subtask.id}
          task={{
            ...subtask,
            deadlineType: null,
            deadlineDate: null,
            deadlineDays: null,
            parentId: task.id,
          }}
          isSubtask
        />
      ))}
    </div>
  )
}
