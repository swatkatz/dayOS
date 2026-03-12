import { useState } from 'react'
import { CATEGORY_COLORS } from '../../constants'

interface Block {
  id: string
  time: string
  duration: number
  title: string
  category: string
  notes: string | null
  skipped: boolean
}

interface Props {
  block: Block
  onSkip?: (blockId: string) => void
  onUpdateDuration?: (blockId: string, duration: number) => void
  readOnly?: boolean
}

function formatTime12h(time: string): string {
  const [h, m] = time.split(':').map(Number)
  const ampm = h >= 12 ? 'PM' : 'AM'
  const hour12 = h === 0 ? 12 : h > 12 ? h - 12 : h
  return `${hour12}:${String(m).padStart(2, '0')} ${ampm}`
}

export default function BlockCard({ block, onSkip, onUpdateDuration, readOnly }: Props) {
  const [editing, setEditing] = useState(false)
  const [draftDuration, setDraftDuration] = useState(String(block.duration))
  const [validationError, setValidationError] = useState<string | null>(null)

  const color = CATEGORY_COLORS[block.category] || '#6b7280'

  const handleDurationClick = () => {
    if (block.skipped || readOnly) return
    setDraftDuration(String(block.duration))
    setValidationError(null)
    setEditing(true)
  }

  const handleDurationSubmit = () => {
    const val = parseInt(draftDuration, 10)
    if (isNaN(val) || val <= 0) {
      setValidationError('Must be > 0')
      return
    }
    setValidationError(null)
    setEditing(false)
    if (val !== block.duration) {
      onUpdateDuration?.(block.id, val)
    }
  }

  const handleDurationKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleDurationSubmit()
    if (e.key === 'Escape') {
      setEditing(false)
      setValidationError(null)
    }
  }

  return (
    <div
      className={`relative rounded-lg p-4 bg-bg-surface ${block.skipped ? 'opacity-50' : ''}`}
      style={{ borderLeft: `4px solid ${color}` }}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3 mb-1">
            <span className="text-text-secondary text-sm font-mono">
              {formatTime12h(block.time)}
            </span>

            {editing ? (
              <div className="flex items-center gap-1">
                <input
                  type="number"
                  value={draftDuration}
                  onChange={(e) => setDraftDuration(e.target.value)}
                  onBlur={handleDurationSubmit}
                  onKeyDown={handleDurationKeyDown}
                  autoFocus
                  className="w-16 bg-bg-surface-hover border border-border-default rounded px-2 py-0.5 text-sm text-text-primary focus:border-accent outline-none"
                />
                <span className="text-text-secondary text-sm">min</span>
                {validationError && (
                  <span className="text-red-400 text-xs">{validationError}</span>
                )}
              </div>
            ) : (
              <button
                onClick={handleDurationClick}
                className={`text-xs px-2 py-0.5 rounded bg-bg-surface-hover text-text-secondary ${
                  !block.skipped && !readOnly ? 'hover:text-text-primary cursor-pointer' : 'cursor-default'
                }`}
              >
                {block.duration} min
              </button>
            )}
          </div>

          <p className={`font-medium text-text-primary ${block.skipped ? 'line-through' : ''}`}>
            {block.title}
          </p>

          <span className="text-xs mt-1 inline-block" style={{ color }}>
            {block.category.toLowerCase()}
          </span>

          {block.notes && !block.skipped && (
            <p className="text-text-secondary text-sm mt-1">{block.notes}</p>
          )}
        </div>

        <div className="flex-shrink-0">
          {block.skipped ? (
            <span className="text-text-secondary text-xs">Skipped</span>
          ) : !readOnly && onSkip ? (
            <button
              onClick={() => onSkip(block.id)}
              className="text-text-secondary hover:text-red-400 text-sm transition-colors"
              title="Skip block"
            >
              ✕
            </button>
          ) : null}
        </div>
      </div>
    </div>
  )
}
