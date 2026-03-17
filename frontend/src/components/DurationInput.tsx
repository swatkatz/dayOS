import { useState, useRef } from 'react'

const PRESETS = [15, 30, 45, 60, 90, 120]

interface Props {
  value: number
  onChange: (minutes: number) => void
  className?: string
}

export default function DurationInput({ value, onChange, className = '' }: Props) {
  const isPreset = PRESETS.includes(value)
  const [custom, setCustom] = useState(!isPreset)
  const [draft, setDraft] = useState(String(value))
  const inputRef = useRef<HTMLInputElement>(null)

  const base = 'bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary focus:border-accent outline-none'

  if (!custom && isPreset) {
    return (
      <div className={`flex items-center gap-2 ${className}`}>
        <select
          value={String(value)}
          onChange={(e) => {
            const v = e.target.value
            if (v === 'custom') {
              setCustom(true)
              setDraft(String(value))
              setTimeout(() => inputRef.current?.select(), 0)
            } else {
              onChange(parseInt(v))
            }
          }}
          className={`w-28 ${base}`}
        >
          {PRESETS.map((p) => (
            <option key={p} value={String(p)}>{p} min</option>
          ))}
          <option value="custom">Custom...</option>
        </select>
      </div>
    )
  }

  return (
    <div className={`flex items-center gap-2 ${className}`}>
      <input
        ref={inputRef}
        type="number"
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onFocus={(e) => e.target.select()}
        onBlur={() => {
          const v = parseInt(draft)
          if (!isNaN(v) && v > 0) {
            onChange(v)
            if (PRESETS.includes(v)) setCustom(false)
          } else {
            setDraft(String(value))
          }
        }}
        onKeyDown={(e) => {
          if (e.key === 'Enter') (e.target as HTMLInputElement).blur()
          if (e.key === 'Escape') {
            setDraft(String(value))
            if (PRESETS.includes(value)) setCustom(false)
          }
        }}
        autoFocus
        min={1}
        className={`w-20 ${base}`}
      />
      <span className="text-text-secondary text-sm">min</span>
    </div>
  )
}
