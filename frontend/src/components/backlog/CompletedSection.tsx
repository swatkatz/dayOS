import { useState } from 'react'
import TaskRow from './TaskRow'

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
}

interface Props {
  tasks: Task[]
}

export default function CompletedSection({ tasks }: Props) {
  const [open, setOpen] = useState(false)

  if (tasks.length === 0) return null

  return (
    <div>
      <button
        onClick={() => setOpen(!open)}
        className="w-full text-left text-sm font-medium text-text-secondary uppercase tracking-wide border-b border-border-default pb-2 mb-3 hover:text-text-primary transition-colors"
      >
        Completed ({tasks.length}) {open ? '▾' : '▸'}
      </button>

      {open && (
        <div>
          {tasks.map((task) => (
            <TaskRow key={task.id} task={task} readOnly />
          ))}
        </div>
      )}
    </div>
  )
}
