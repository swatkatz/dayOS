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
}

interface Props {
  blocks: Block[]
  onAccept: (blocks: Block[]) => void
  accepting: boolean
}

function generateId(): string {
  return crypto.randomUUID()
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

  const handleMove = (blockId: string, direction: 'up' | 'down') => {
    setLocalBlocks((prev) => {
      const sorted = [...prev].sort((a, b) => a.time.localeCompare(b.time))
      const idx = sorted.findIndex((b) => b.id === blockId)
      if (idx < 0) return prev
      const swapIdx = direction === 'up' ? idx - 1 : idx + 1
      if (swapIdx < 0 || swapIdx >= sorted.length) return prev
      const newBlocks = sorted.map((b) => ({ ...b }))
      const tmpTime = newBlocks[idx].time
      newBlocks[idx].time = newBlocks[swapIdx].time
      newBlocks[swapIdx].time = tmpTime
      return newBlocks
    })
  }

  const handleAdd = () => {
    const newBlock: Block = {
      id: generateId(),
      time: '09:00',
      duration: 30,
      title: 'New block',
      category: 'ADMIN',
      taskId: null,
      routineId: null,
      notes: null,
      skipped: false,
    }
    setLocalBlocks((prev) => [...prev, newBlock])
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
          onMove={handleMove}
        />
        <button
          onClick={handleAdd}
          className="w-full mt-2 py-2 border border-dashed border-border-default rounded text-text-secondary hover:text-text-primary hover:border-accent text-sm transition-colors"
        >
          + Add block
        </button>
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
