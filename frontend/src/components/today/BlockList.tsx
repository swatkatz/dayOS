import BlockCard from './BlockCard'
import NowIndicator, { useNowPosition } from './NowIndicator'

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
  onMove?: (blockId: string, direction: 'up' | 'down') => void
  readOnly?: boolean
  showNow?: boolean
}

export default function BlockList({ blocks, onSkip, onUpdateDuration, onUpdateBlock, onDelete, onMove, readOnly, showNow }: Props) {
  const sorted = [...blocks].sort((a, b) => a.time.localeCompare(b.time))
  const nowPosition = useNowPosition(sorted)

  return (
    <div className="space-y-2">
      {sorted.map((block, i) => (
        <div key={block.id}>
          {showNow && nowPosition === i && <NowIndicator blocks={sorted} />}
          <BlockCard
            block={block}
            onSkip={onSkip}
            onUpdateDuration={onUpdateDuration}
            onUpdateBlock={onUpdateBlock}
            onDelete={onDelete}
            onMoveUp={onMove && i > 0 ? () => onMove(block.id, 'up') : undefined}
            onMoveDown={onMove && i < sorted.length - 1 ? () => onMove(block.id, 'down') : undefined}
            readOnly={readOnly}
          />
        </div>
      ))}
      {showNow && nowPosition === sorted.length && <NowIndicator blocks={sorted} />}
    </div>
  )
}
