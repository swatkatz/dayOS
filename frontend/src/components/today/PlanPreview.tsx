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
  onAccept: () => void
  accepting: boolean
}

export default function PlanPreview({ blocks, onAccept, accepting }: Props) {
  if (blocks.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-text-secondary">
        Send a message to generate your plan
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-y-auto p-4">
        <BlockList blocks={blocks} readOnly />
      </div>
      <div className="p-4 border-t border-border-default">
        <button
          onClick={onAccept}
          disabled={blocks.length === 0 || accepting}
          className="w-full py-2 px-4 bg-accent text-black rounded font-medium hover:bg-accent-hover disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        >
          {accepting ? 'Accepting...' : 'Accept Plan'}
        </button>
      </div>
    </div>
  )
}
