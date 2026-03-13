import { useState, useRef } from 'react'
import BlockCard from './BlockCard'
import NowIndicator, { useNowPosition, useActiveBlockId } from './NowIndicator'

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
  onSkip?: (blockId: string) => void
  onUpdateDuration?: (blockId: string, duration: number) => void
  onUpdateBlock?: (blockId: string, updates: Partial<Block>) => void
  onDelete?: (blockId: string) => void
  onReorder?: (reorderedBlocks: Block[]) => void
  readOnly?: boolean
  showNow?: boolean
}

export default function BlockList({ blocks, onSkip, onUpdateDuration, onUpdateBlock, onDelete, onReorder, readOnly, showNow }: Props) {
  const sorted = [...blocks].sort((a, b) => a.time.localeCompare(b.time))
  const nowPosition = useNowPosition(sorted)
  const activeBlockId = useActiveBlockId(sorted)
  const [draggedId, setDraggedId] = useState<string | null>(null)
  const [dropTargetIdx, setDropTargetIdx] = useState<number | null>(null)
  const dragCounter = useRef(0)

  const handleDragStart = (e: React.DragEvent, blockId: string) => {
    setDraggedId(blockId)
    e.dataTransfer.effectAllowed = 'move'
    // Make the drag image slightly transparent
    if (e.currentTarget instanceof HTMLElement) {
      e.dataTransfer.setDragImage(e.currentTarget, 0, 0)
    }
  }

  const handleDragEnd = () => {
    setDraggedId(null)
    setDropTargetIdx(null)
    dragCounter.current = 0
  }

  const handleDragEnter = (e: React.DragEvent, idx: number) => {
    e.preventDefault()
    dragCounter.current++
    setDropTargetIdx(idx)
  }

  const handleDragLeave = () => {
    dragCounter.current--
    if (dragCounter.current <= 0) {
      setDropTargetIdx(null)
      dragCounter.current = 0
    }
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
  }

  const handleDrop = (e: React.DragEvent, targetIdx: number) => {
    e.preventDefault()
    setDropTargetIdx(null)
    dragCounter.current = 0
    if (!draggedId || !onReorder) return

    const fromIdx = sorted.findIndex((b) => b.id === draggedId)
    if (fromIdx < 0 || fromIdx === targetIdx) return

    // Reorder: remove dragged block and insert at target position
    const reordered = [...sorted]
    const [moved] = reordered.splice(fromIdx, 1)
    reordered.splice(targetIdx, 0, moved)

    // Recalculate times sequentially: first block keeps its time, each subsequent starts after the previous ends
    const startTime = reordered[0].time
    const [startH, startM] = startTime.split(':').map(Number)
    let currentMinutes = startH * 60 + startM

    const withTimes = reordered.map((block) => {
      const h = Math.floor(currentMinutes / 60)
      const m = currentMinutes % 60
      const newTime = `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
      currentMinutes += block.duration
      return { ...block, time: newTime }
    })

    onReorder(withTimes)
    setDraggedId(null)
  }

  const draggable = !readOnly && !!onReorder

  return (
    <div className="space-y-2">
      {sorted.map((block, i) => (
        <div
          key={block.id}
          draggable={draggable && !block.skipped}
          onDragStart={(e) => handleDragStart(e, block.id)}
          onDragEnd={handleDragEnd}
          onDragEnter={(e) => handleDragEnter(e, i)}
          onDragLeave={handleDragLeave}
          onDragOver={handleDragOver}
          onDrop={(e) => handleDrop(e, i)}
          className={`${draggable && !block.skipped ? 'cursor-grab active:cursor-grabbing' : ''} ${
            draggedId === block.id ? 'opacity-30' : ''
          } ${
            dropTargetIdx === i && draggedId !== block.id
              ? 'border-t-2 border-accent'
              : ''
          } transition-opacity`}
        >
          {showNow && nowPosition === i && <NowIndicator blocks={sorted} />}
          <BlockCard
            block={block}
            onSkip={onSkip}
            onUpdateDuration={onUpdateDuration}
            onUpdateBlock={onUpdateBlock}
            onDelete={onDelete}
            readOnly={readOnly}
            active={showNow && activeBlockId === block.id}
          />
        </div>
      ))}
      {showNow && nowPosition === sorted.length && <NowIndicator blocks={sorted} />}
    </div>
  )
}
