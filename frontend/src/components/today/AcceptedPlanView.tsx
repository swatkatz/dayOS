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
  onSkip: (blockId: string) => void
  onUpdateDuration: (blockId: string, duration: number) => void
  onReplan: () => void
}

export default function AcceptedPlanView({ blocks, onSkip, onUpdateDuration, onReplan }: Props) {
  return (
    <div className="max-w-2xl mx-auto">
      <div className="mb-6">
        <BlockList
          blocks={blocks}
          onSkip={onSkip}
          onUpdateDuration={onUpdateDuration}
          showNow
        />
      </div>

      <button
        onClick={onReplan}
        className="w-full py-2 px-4 rounded border border-border-default text-text-secondary hover:text-text-primary hover:border-accent transition-colors"
      >
        Something came up
      </button>
    </div>
  )
}
