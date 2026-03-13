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
  onUpdateBlock?: (blockId: string, updates: Partial<Block>) => void
  onDelete?: (blockId: string) => void
  onMoveUp?: () => void
  onMoveDown?: () => void
  readOnly?: boolean
}

function formatTime12h(time: string): string {
  const [h, m] = time.split(':').map(Number)
  const ampm = h >= 12 ? 'PM' : 'AM'
  const hour12 = h === 0 ? 12 : h > 12 ? h - 12 : h
  return `${hour12}:${String(m).padStart(2, '0')} ${ampm}`
}

const CATEGORIES = ['JOB', 'INTERVIEW', 'PROJECT', 'MEAL', 'BABY', 'EXERCISE', 'ADMIN']

export default function BlockCard({ block, onSkip, onUpdateDuration, onUpdateBlock, onDelete, onMoveUp, onMoveDown, readOnly }: Props) {
  const [expanded, setExpanded] = useState(false)
  const [editing, setEditing] = useState(false)
  const [draftDuration, setDraftDuration] = useState(String(block.duration))
  const [validationError, setValidationError] = useState<string | null>(null)

  const color = CATEGORY_COLORS[block.category] || CATEGORY_COLORS[block.category.toUpperCase()] || '#6b7280'
  const editable = !readOnly && !block.skipped && !!onUpdateBlock

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
      if (onUpdateBlock) {
        onUpdateBlock(block.id, { duration: val })
      } else {
        onUpdateDuration?.(block.id, val)
      }
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
    <div className={`rounded-lg bg-bg-surface ${block.skipped ? 'opacity-50' : ''}`} style={{ borderLeft: `4px solid ${color}` }}>
      <div className="relative p-4">
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

          <div className="flex items-center gap-2 flex-shrink-0">
            {block.skipped ? (
              <span className="text-text-secondary text-xs">Skipped</span>
            ) : (
              <>
                {onMoveUp && (
                  <button
                    onClick={onMoveUp}
                    className="text-text-secondary hover:text-text-primary text-sm transition-colors"
                    title="Move up"
                  >
                    ↑
                  </button>
                )}
                {onMoveDown && (
                  <button
                    onClick={onMoveDown}
                    className="text-text-secondary hover:text-text-primary text-sm transition-colors"
                    title="Move down"
                  >
                    ↓
                  </button>
                )}
                {editable && (
                  <button
                    onClick={() => setExpanded(!expanded)}
                    className="text-text-secondary hover:text-text-primary text-sm transition-colors"
                    title="Edit block"
                  >
                    {expanded ? '▲' : '✎'}
                  </button>
                )}
                {!readOnly && onSkip && (
                  <button
                    onClick={() => onSkip(block.id)}
                    className="text-text-secondary hover:text-red-400 text-sm transition-colors"
                    title="Skip block"
                  >
                    ✕
                  </button>
                )}
                {onDelete && (
                  <button
                    onClick={() => onDelete(block.id)}
                    className="text-text-secondary hover:text-red-400 text-sm transition-colors"
                    title="Remove block"
                  >
                    🗑
                  </button>
                )}
              </>
            )}
          </div>
        </div>
      </div>

      {/* Expanded edit panel */}
      {expanded && editable && (
        <div className="px-4 pb-4 pt-0 border-t border-border-default space-y-3 text-sm">
          <div className="flex gap-4 flex-wrap pt-3">
            <label className="flex flex-col gap-1">
              <span className="text-text-secondary text-xs">Title</span>
              <input
                defaultValue={block.title}
                onBlur={(e) => {
                  const val = e.target.value.trim()
                  if (val && val !== block.title) onUpdateBlock(block.id, { title: val })
                }}
                onKeyDown={(e) => { if (e.key === 'Enter') (e.target as HTMLInputElement).blur() }}
                className="bg-bg-primary border border-border-default rounded px-2 py-1 text-text-primary w-64"
              />
            </label>

            <label className="flex flex-col gap-1">
              <span className="text-text-secondary text-xs">Time</span>
              <input
                type="time"
                defaultValue={block.time}
                onBlur={(e) => {
                  if (e.target.value && e.target.value !== block.time) onUpdateBlock(block.id, { time: e.target.value })
                }}
                className="bg-bg-primary border border-border-default rounded px-2 py-1 text-text-primary"
              />
            </label>

            <label className="flex flex-col gap-1">
              <span className="text-text-secondary text-xs">Category</span>
              <select
                value={block.category.toUpperCase()}
                onChange={(e) => onUpdateBlock(block.id, { category: e.target.value })}
                className="bg-bg-primary border border-border-default rounded px-2 py-1 text-text-primary"
              >
                {CATEGORIES.map((c) => <option key={c} value={c}>{c}</option>)}
              </select>
            </label>
          </div>

          <label className="flex flex-col gap-1">
            <span className="text-text-secondary text-xs">Notes</span>
            <textarea
              defaultValue={block.notes ?? ''}
              onBlur={(e) => {
                const val = e.target.value.trim()
                if (val !== (block.notes ?? '')) onUpdateBlock(block.id, { notes: val || null })
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
