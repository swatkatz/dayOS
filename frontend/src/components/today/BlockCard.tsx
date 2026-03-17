import { useState } from 'react'
import { CATEGORY_COLORS } from '../../constants'
import DurationInput from '../DurationInput'

interface Block {
  id: string
  time: string
  duration: number
  title: string
  category: string
  notes: string | null
  skipped: boolean
  done: boolean
}

interface Props {
  block: Block
  onSkip?: (blockId: string) => void
  onUnskip?: (blockId: string) => void
  onComplete?: (blockId: string) => void
  onUpdateDuration?: (blockId: string, duration: number) => void
  onUpdateBlock?: (blockId: string, updates: Partial<Block>) => void
  onDelete?: (blockId: string) => void
  readOnly?: boolean
  active?: boolean
}

function formatTime12h(time: string): string {
  const [h, m] = time.split(':').map(Number)
  const ampm = h >= 12 ? 'PM' : 'AM'
  const hour12 = h === 0 ? 12 : h > 12 ? h - 12 : h
  return `${hour12}:${String(m).padStart(2, '0')} ${ampm}`
}

function endTime(time: string, duration: number): string {
  const [h, m] = time.split(':').map(Number)
  const total = h * 60 + m + duration
  return formatTime12h(`${String(Math.floor(total / 60)).padStart(2, '0')}:${String(total % 60).padStart(2, '0')}`)
}

const CATEGORIES = ['JOB', 'INTERVIEW', 'PROJECT', 'MEAL', 'BABY', 'EXERCISE', 'ADMIN']

export default function BlockCard({ block, onSkip, onUnskip, onComplete, onUpdateDuration, onUpdateBlock, onDelete, readOnly, active }: Props) {
  const [expanded, setExpanded] = useState(false)
  const [editing, setEditing] = useState(false)

  const color = CATEGORY_COLORS[block.category] || CATEGORY_COLORS[block.category.toUpperCase()] || '#6b7280'
  const editable = !readOnly && !block.skipped && !!onUpdateBlock

  const handleDurationClick = () => {
    if (block.skipped || readOnly) return
    setEditing(true)
  }

  return (
    <div
      className={`rounded-xl bg-bg-surface transition-all ${block.skipped ? 'opacity-40' : ''} ${
        active
          ? 'ring-2 ring-accent shadow-[0_0_20px_rgba(197,165,90,0.15)]'
          : 'hover:shadow-[0_2px_12px_rgba(0,0,0,0.3)]'
      }`}
      style={{ borderLeft: `3px solid ${color}` }}
    >
      <div className="relative p-3 md:p-4">
        <div className="flex items-start justify-between gap-2 md:gap-3">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 md:gap-3 mb-1 flex-wrap">
              <span className="text-text-secondary text-xs md:text-sm font-mono whitespace-nowrap">
                {formatTime12h(block.time)} – {endTime(block.time, block.duration)}
              </span>

              {editing ? (
                <DurationInput
                  value={block.duration}
                  onChange={(v) => {
                    setEditing(false)
                    if (v !== block.duration) {
                      if (onUpdateBlock) {
                        onUpdateBlock(block.id, { duration: v })
                      } else {
                        onUpdateDuration?.(block.id, v)
                      }
                    }
                  }}
                />
              ) : (
                <button
                  onClick={handleDurationClick}
                  className={`text-xs px-2 py-0.5 rounded-full bg-bg-surface-hover text-text-secondary ${
                    !block.skipped && !readOnly ? 'hover:text-text-primary cursor-pointer' : 'cursor-default'
                  }`}
                >
                  {block.duration}m
                </button>
              )}
            </div>

            <p className={`font-medium text-[15px] text-text-primary leading-snug ${block.skipped ? 'line-through text-text-secondary' : ''}`}>
              {block.title}
            </p>

            <span className="text-xs mt-1.5 inline-block font-medium opacity-80" style={{ color }}>
              {block.category.toLowerCase()}
            </span>

            {block.notes && !block.skipped && (
              <p className="text-text-secondary text-sm mt-1.5 leading-relaxed">{block.notes}</p>
            )}
          </div>

          {/* Action buttons — larger touch targets on mobile */}
          <div className="flex items-center gap-1 md:gap-2 flex-shrink-0">
            {block.skipped ? (
              <>
                <span className="text-text-secondary text-xs">Skipped</span>
                {!readOnly && onUnskip && (
                  <button
                    onClick={() => onUnskip(block.id)}
                    className="text-text-secondary hover:text-accent text-xs p-2 -m-1 transition-colors"
                    title="Undo skip"
                  >
                    Undo
                  </button>
                )}
              </>
            ) : (
              <>
                {!readOnly && onComplete && (
                  <button
                    onClick={() => onComplete(block.id)}
                    className="text-text-secondary hover:text-emerald-400 active:text-emerald-400 active:scale-125 text-lg p-2 -m-1 transition-all"
                    title="Mark as done"
                  >
                    &#x2713;
                  </button>
                )}
                {editable && (
                  <button
                    onClick={() => setExpanded(!expanded)}
                    className="text-text-secondary hover:text-text-primary active:text-text-primary text-sm p-2 -m-1 transition-colors"
                    title="Edit block"
                  >
                    {expanded ? '▲' : '✎'}
                  </button>
                )}
                {!readOnly && onSkip && (
                  <button
                    onClick={() => onSkip(block.id)}
                    className="text-text-secondary hover:text-red-400 active:text-red-400 text-sm p-2 -m-1 transition-colors"
                    title="Skip block"
                  >
                    ✕
                  </button>
                )}
                {onDelete && (
                  <button
                    onClick={() => onDelete(block.id)}
                    className="text-text-secondary hover:text-red-400 active:text-red-400 text-sm p-2 -m-1 transition-colors"
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
        <div className="px-3 md:px-4 pb-3 md:pb-4 pt-0 border-t border-border-subtle space-y-3 text-sm">
          <div className="flex gap-3 md:gap-4 flex-wrap pt-3">
            <label className="flex flex-col gap-1 flex-1 min-w-[200px]">
              <span className="text-text-secondary text-xs">Title</span>
              <input
                defaultValue={block.title}
                onBlur={(e) => {
                  const val = e.target.value.trim()
                  if (val && val !== block.title) onUpdateBlock(block.id, { title: val })
                }}
                onKeyDown={(e) => { if (e.key === 'Enter') (e.target as HTMLInputElement).blur() }}
                className="bg-bg-primary border border-border-default rounded-lg px-3 py-2 text-text-primary"
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
                className="bg-bg-primary border border-border-default rounded-lg px-3 py-2 text-text-primary"
              />
            </label>

            <label className="flex flex-col gap-1">
              <span className="text-text-secondary text-xs">Category</span>
              <select
                value={block.category.toUpperCase()}
                onChange={(e) => onUpdateBlock(block.id, { category: e.target.value })}
                className="bg-bg-primary border border-border-default rounded-lg px-3 py-2 text-text-primary"
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
              className="bg-bg-primary border border-border-default rounded-lg px-3 py-2 text-text-primary resize-y"
              placeholder="Add notes..."
            />
          </label>
        </div>
      )}
    </div>
  )
}
