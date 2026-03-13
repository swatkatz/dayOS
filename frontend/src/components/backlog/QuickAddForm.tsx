import { useState } from 'react'
import { useMutation } from '@apollo/client/react'
import { CREATE_TASK, GET_TASKS } from '../../graphql/backlog'

interface Props {
  onClose: () => void
}

const CATEGORIES = ['JOB', 'INTERVIEW', 'PROJECT', 'MEAL', 'BABY', 'EXERCISE', 'ADMIN']
const PRIORITIES = ['HIGH', 'MEDIUM', 'LOW']

export default function QuickAddForm({ onClose }: Props) {
  const [title, setTitle] = useState('')
  const [notes, setNotes] = useState('')
  const [category, setCategory] = useState('ADMIN')
  const [priority, setPriority] = useState('MEDIUM')
  const [estimatedMinutes, setEstimatedMinutes] = useState(60)

  const [createTask, { loading }] = useMutation(CREATE_TASK, {
    refetchQueries: [{ query: GET_TASKS, variables: { includeCompleted: true } }],
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!title.trim()) return

    await createTask({
      variables: {
        input: {
          title: title.trim(),
          category,
          priority,
          estimatedMinutes,
          ...(notes.trim() ? { notes: notes.trim() } : {}),
        },
      },
    })
    onClose()
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') onClose()
  }

  return (
    <form
      onSubmit={handleSubmit}
      onKeyDown={handleKeyDown}
      className="bg-bg-surface rounded-lg p-4 border border-border-default mb-4"
    >
      <div className="grid grid-cols-2 gap-3 mb-3">
        <input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Task title"
          autoFocus
          className="col-span-2 bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary placeholder:text-text-secondary focus:border-accent focus:ring-1 focus:ring-accent outline-none"
        />
        <textarea
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          placeholder="Notes / description (optional)"
          rows={2}
          className="col-span-2 bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary placeholder:text-text-secondary focus:border-accent focus:ring-1 focus:ring-accent outline-none resize-y"
        />

        <select
          value={category}
          onChange={(e) => setCategory(e.target.value)}
          className="bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary focus:border-accent outline-none"
        >
          {CATEGORIES.map((c) => (
            <option key={c} value={c}>{c.toLowerCase()}</option>
          ))}
        </select>

        <select
          value={priority}
          onChange={(e) => setPriority(e.target.value)}
          className="bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary focus:border-accent outline-none"
        >
          {PRIORITIES.map((p) => (
            <option key={p} value={p}>{p.toLowerCase()}</option>
          ))}
        </select>

        <div className="flex items-center gap-2">
          <input
            type="number"
            value={estimatedMinutes}
            onChange={(e) => setEstimatedMinutes(parseInt(e.target.value) || 60)}
            min={1}
            className="w-20 bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary focus:border-accent outline-none"
          />
          <span className="text-text-secondary text-sm">min</span>
        </div>
      </div>

      <div className="flex gap-2 justify-end">
        <button
          type="button"
          onClick={onClose}
          className="px-3 py-1.5 text-text-secondary hover:text-text-primary transition-colors"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={!title.trim() || loading}
          className="px-4 py-1.5 bg-accent text-black rounded font-medium hover:bg-accent-hover disabled:opacity-40 transition-colors"
        >
          {loading ? 'Adding...' : 'Add Task'}
        </button>
      </div>
    </form>
  )
}
