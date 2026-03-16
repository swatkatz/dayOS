import { useState } from 'react'
import { useMutation } from '@apollo/client/react'
import { RESOLVE_SKIPPED_BLOCK } from '../../graphql/today'
import { CATEGORY_COLORS } from '../../constants'

interface SkippedBlock {
  id: string
  title: string
  category: string
  taskId: string | null
}

interface Props {
  planId: string
  blocks: SkippedBlock[]
  onDone: () => void
}

export default function SkippedTasksReview({ planId, blocks, onDone }: Props) {
  const [resolved, setResolved] = useState<Set<string>>(new Set())
  const [resolveSkipped] = useMutation(RESOLVE_SKIPPED_BLOCK)

  const handleResolve = async (blockId: string, intentional: boolean) => {
    try {
      await resolveSkipped({
        variables: { planId, blockId, intentional },
      })
      setResolved((prev) => new Set(prev).add(blockId))
    } catch {
      // Mutation error — leave unresolved
    }
  }

  const allResolved = blocks.every((b) => resolved.has(b.id))

  return (
    <div className="flex items-center justify-center min-h-[60vh] px-3">
      <div className="bg-bg-surface rounded-2xl p-5 md:p-6 max-w-lg w-full shadow-[0_4px_24px_rgba(0,0,0,0.4)]">
        <h2 className="text-lg md:text-xl font-semibold text-text-primary mb-1">
          Yesterday's skipped blocks
        </h2>
        <p className="text-text-secondary text-sm mb-5">
          Were these skips intentional? This helps improve future plans.
        </p>

        <div className="space-y-3">
          {blocks.map((block) => {
            const isResolved = resolved.has(block.id)
            return (
              <div
                key={block.id}
                className={`p-3 md:p-4 rounded-xl border border-border-default transition-opacity ${isResolved ? 'opacity-40' : ''}`}
              >
                <div className="flex items-center gap-2 mb-3">
                  <span
                    className="w-2.5 h-2.5 rounded-full flex-shrink-0"
                    style={{ backgroundColor: CATEGORY_COLORS[block.category] || '#6b7280' }}
                  />
                  <span className="text-text-primary font-medium text-[15px]">{block.title}</span>
                </div>

                {isResolved ? (
                  <span className="text-text-secondary text-sm">Resolved</span>
                ) : (
                  <div className="flex items-center gap-2">
                    <span className="text-text-secondary text-sm mr-1">Intentional?</span>
                    <button
                      onClick={() => handleResolve(block.id, true)}
                      className="px-4 py-1.5 text-sm rounded-lg bg-bg-surface-hover text-text-primary hover:bg-accent hover:text-black active:scale-95 transition-all"
                    >
                      Yes
                    </button>
                    <button
                      onClick={() => handleResolve(block.id, false)}
                      className="px-4 py-1.5 text-sm rounded-lg bg-bg-surface-hover text-text-primary hover:bg-accent hover:text-black active:scale-95 transition-all"
                    >
                      No
                    </button>
                  </div>
                )}
              </div>
            )
          })}
        </div>

        <button
          onClick={onDone}
          disabled={!allResolved}
          className="mt-6 w-full py-2.5 px-4 rounded-xl font-semibold transition-all disabled:opacity-40 disabled:cursor-not-allowed bg-accent text-black hover:bg-accent-hover active:scale-[0.99]"
        >
          Done — Start Planning
        </button>
      </div>
    </div>
  )
}
