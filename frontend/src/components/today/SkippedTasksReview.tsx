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
    <div className="flex items-center justify-center min-h-[60vh]">
      <div className="bg-bg-surface rounded-lg p-6 max-w-lg w-full">
        <h2 className="text-xl font-semibold text-text-primary mb-4">
          Yesterday you skipped {blocks.length} block{blocks.length !== 1 ? 's' : ''}:
        </h2>

        <div className="space-y-4">
          {blocks.map((block) => {
            const isResolved = resolved.has(block.id)
            return (
              <div
                key={block.id}
                className={`p-4 rounded-lg border border-border-default ${isResolved ? 'opacity-50' : ''}`}
              >
                <div className="flex items-center gap-2 mb-3">
                  <span
                    className="w-2.5 h-2.5 rounded-full flex-shrink-0"
                    style={{ backgroundColor: CATEGORY_COLORS[block.category] || '#6b7280' }}
                  />
                  <span className="text-text-primary font-medium">{block.title}</span>
                </div>

                {isResolved ? (
                  <span className="text-text-secondary text-sm">Resolved</span>
                ) : (
                  <div className="flex items-center gap-2">
                    <span className="text-text-secondary text-sm mr-2">Intentional?</span>
                    <button
                      onClick={() => handleResolve(block.id, true)}
                      className="px-3 py-1 text-sm rounded bg-bg-surface-hover text-text-primary hover:bg-accent hover:text-black transition-colors"
                    >
                      Yes
                    </button>
                    <button
                      onClick={() => handleResolve(block.id, false)}
                      className="px-3 py-1 text-sm rounded bg-bg-surface-hover text-text-primary hover:bg-accent hover:text-black transition-colors"
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
          className="mt-6 w-full py-2 px-4 rounded font-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed bg-accent text-black hover:bg-accent-hover"
        >
          Done — Start Planning
        </button>
      </div>
    </div>
  )
}
