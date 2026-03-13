import { useEffect, useRef, useState } from 'react'

interface Block {
  time: string
}

interface Props {
  blocks: Block[]
}

function getBlockDate(time: string): Date {
  const [h, m] = time.split(':').map(Number)
  const now = new Date()
  return new Date(now.getFullYear(), now.getMonth(), now.getDate(), h, m)
}

export function useNowPosition(blocks: Block[]): number {
  const [position, setPosition] = useState(() => computePosition(blocks))

  useEffect(() => {
    setPosition(computePosition(blocks))
    const interval = setInterval(() => {
      setPosition(computePosition(blocks))
    }, 60_000)
    return () => clearInterval(interval)
  }, [blocks])

  return position
}

interface ActiveBlock {
  id: string
  time: string
  duration: number
  skipped: boolean
}

function computePosition(blocks: Block[]): number {
  if (blocks.length === 0) return -1
  const now = new Date()
  for (let i = 0; i < blocks.length; i++) {
    const blockTime = getBlockDate(blocks[i].time)
    if (now < blockTime) return i
  }
  return blocks.length
}

function computeActiveBlockId(blocks: ActiveBlock[]): string | null {
  if (blocks.length === 0) return null
  const now = new Date()
  for (let i = 0; i < blocks.length; i++) {
    const b = blocks[i]
    if (b.skipped) continue
    const start = getBlockDate(b.time)
    const end = new Date(start.getTime() + b.duration * 60_000)
    if (now >= start && now < end) return b.id
  }
  return null
}

export function useActiveBlockId(blocks: ActiveBlock[]): string | null {
  const [activeId, setActiveId] = useState<string | null>(() => computeActiveBlockId(blocks))

  useEffect(() => {
    setActiveId(computeActiveBlockId(blocks))
    const interval = setInterval(() => {
      setActiveId(computeActiveBlockId(blocks))
    }, 60_000)
    return () => clearInterval(interval)
  }, [blocks])

  return activeId
}

export default function NowIndicator({ blocks }: Props) {
  const position = useNowPosition(blocks)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    ref.current?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }, [position])

  if (position < 0) return null

  return (
    <div ref={ref} className="flex items-center gap-2 py-1" data-position={position}>
      <span className="text-xs font-bold text-accent">NOW</span>
      <div className="flex-1 border-t-2 border-dashed border-accent" />
    </div>
  )
}
