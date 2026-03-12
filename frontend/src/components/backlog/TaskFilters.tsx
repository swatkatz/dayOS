const CATEGORIES = ['All', 'JOB', 'INTERVIEW', 'PROJECT', 'MEAL', 'BABY', 'EXERCISE', 'ADMIN']
const PRIORITIES = ['All', 'HIGH', 'MEDIUM', 'LOW']

interface Props {
  category: string
  priority: string
  onCategoryChange: (value: string) => void
  onPriorityChange: (value: string) => void
}

export default function TaskFilters({ category, priority, onCategoryChange, onPriorityChange }: Props) {
  return (
    <div className="flex gap-3 items-center">
      <span className="text-text-secondary text-sm">Filters:</span>
      <select
        value={category}
        onChange={(e) => onCategoryChange(e.target.value)}
        className="bg-bg-surface border border-border-default rounded px-3 py-1.5 text-text-primary text-sm focus:border-accent outline-none"
      >
        {CATEGORIES.map((c) => (
          <option key={c} value={c}>{c === 'All' ? 'All categories' : c.toLowerCase()}</option>
        ))}
      </select>
      <select
        value={priority}
        onChange={(e) => onPriorityChange(e.target.value)}
        className="bg-bg-surface border border-border-default rounded px-3 py-1.5 text-text-primary text-sm focus:border-accent outline-none"
      >
        {PRIORITIES.map((p) => (
          <option key={p} value={p}>{p === 'All' ? 'All priorities' : p.toLowerCase()}</option>
        ))}
      </select>
    </div>
  )
}
