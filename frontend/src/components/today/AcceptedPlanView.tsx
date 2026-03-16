import { useState, useCallback } from 'react'
import BlockList from './BlockList'
import Confetti from './Confetti'
import { useDailyMotivation } from '../../hooks/useDailyMotivation'

interface Block {
  id: string
  time: string
  duration: number
  title: string
  category: string
  taskId: string | null
  routineId: string | null
  notes: string | null
  skipped: boolean
  done: boolean
}

interface Props {
  blocks: Block[]
  onSkip: (blockId: string) => void
  onUnskip: (blockId: string) => void
  onComplete: (blockId: string) => void
  onUpdateDuration: (blockId: string, duration: number) => void
  onReplan: () => void
}

export default function AcceptedPlanView({ blocks, onSkip, onUnskip, onComplete, onUpdateDuration, onReplan }: Props) {
  const total = blocks.length
  const doneCount = blocks.filter((b) => b.done).length
  const skippedCount = blocks.filter((b) => b.skipped).length
  const activeCount = total - skippedCount
  const progress = activeCount > 0 ? Math.round((doneCount / activeCount) * 100) : 0
  const motivation = useDailyMotivation()

  const [showConfetti, setShowConfetti] = useState(false)

  const handleComplete = useCallback((blockId: string) => {
    setShowConfetti(true)
    onComplete(blockId)
  }, [onComplete])

  return (
    <div className="max-w-2xl mx-auto">
      {showConfetti && (
        <Confetti onDone={() => setShowConfetti(false)} />
      )}

      {/* Progress header */}
      <div className="mb-4 md:mb-6">
        <div className="flex items-center justify-between mb-2">
          <h1 className="text-lg md:text-xl font-semibold text-text-primary">Today's Plan</h1>
          <span className="text-sm text-text-secondary">
            {doneCount}/{activeCount} done
          </span>
        </div>
        <div className="h-1 bg-bg-surface rounded-full overflow-hidden">
          <div
            className="h-full bg-accent rounded-full transition-all duration-500"
            style={{ width: `${progress}%` }}
          />
        </div>
        <p className="text-text-secondary text-xs mt-2 italic">{motivation}</p>
      </div>

      <div className="mb-6">
        <BlockList
          blocks={blocks}
          onSkip={onSkip}
          onUnskip={onUnskip}
          onComplete={handleComplete}
          onUpdateDuration={onUpdateDuration}
          showNow
        />
      </div>

      <button
        onClick={onReplan}
        className="w-full py-2.5 px-4 rounded-xl border border-border-default text-text-secondary hover:text-text-primary hover:border-accent active:scale-[0.99] transition-all"
      >
        Something came up
      </button>
    </div>
  )
}
