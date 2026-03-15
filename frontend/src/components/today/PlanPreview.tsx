import { useState, useEffect } from 'react'
import BlockList from './BlockList'

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
  onAccept: (blocks: Block[]) => void
  accepting: boolean
}

export default function PlanPreview({ blocks: sourceBlocks, onAccept, accepting }: Props) {
  const [localBlocks, setLocalBlocks] = useState<Block[]>(sourceBlocks)

  // Sync when AI generates new blocks
  useEffect(() => {
    setLocalBlocks(sourceBlocks)
  }, [sourceBlocks])

  const handleUpdateBlock = (blockId: string, updates: Partial<Block>) => {
    setLocalBlocks((prev) =>
      prev.map((b) => (b.id === blockId ? { ...b, ...updates } : b))
    )
  }

  const handleDelete = (blockId: string) => {
    setLocalBlocks((prev) => prev.filter((b) => b.id !== blockId))
  }

  const handleReorder = (reorderedBlocks: Block[]) => {
    setLocalBlocks(reorderedBlocks)
  }

  if (localBlocks.length === 0 && sourceBlocks.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-text-secondary">
        Send a message to generate your plan
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-y-auto p-4">
        <BlockList
          blocks={localBlocks}
          onUpdateBlock={handleUpdateBlock}
          onDelete={handleDelete}
          onReorder={handleReorder}
        />
      </div>
      <div className="p-4 border-t border-border-default">
        <button
          onClick={() => onAccept(localBlocks)}
          disabled={localBlocks.length === 0 || accepting}
          className="w-full py-2 px-4 bg-accent text-black rounded font-medium hover:bg-accent-hover disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        >
          {accepting ? 'Accepting...' : 'Accept Plan'}
        </button>
      </div>
    </div>
  )
}
