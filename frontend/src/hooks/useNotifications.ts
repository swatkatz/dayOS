import { useEffect, useRef, useCallback } from 'react'

interface Block {
  id: string
  time: string
  duration: number
  title: string
  skipped: boolean
}

const PERMISSION_KEY = 'dayos_notif_denied'

export function useNotifications(blocks: Block[], isAccepted: boolean) {
  const timeoutIds = useRef<number[]>([])

  const clearAll = useCallback(() => {
    timeoutIds.current.forEach((id) => clearTimeout(id))
    timeoutIds.current = []
  }, [])

  useEffect(() => {
    if (typeof Notification === 'undefined') return
    if (Notification.permission === 'default' && !localStorage.getItem(PERMISSION_KEY)) {
      Notification.requestPermission().then((perm) => {
        if (perm === 'denied') {
          localStorage.setItem(PERMISSION_KEY, 'true')
        }
      })
    }
  }, [])

  useEffect(() => {
    clearAll()

    if (!isAccepted || Notification.permission !== 'granted') return

    const now = new Date()
    const today = now.toISOString().slice(0, 10)

    for (const block of blocks) {
      if (block.skipped) continue

      const [hours, minutes] = block.time.split(':').map(Number)
      const blockTime = new Date(`${today}T${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:00`)
      const delay = blockTime.getTime() - now.getTime()

      if (delay > 0) {
        const id = window.setTimeout(() => {
          new Notification(`Time for: ${block.title} (${block.duration} min)`, {
            body: 'Click to focus the app',
          })
        }, delay)
        timeoutIds.current.push(id)
      }
    }

    // End-of-day nudge at 16:00
    const fourPM = new Date(`${today}T16:00:00`)
    const nudgeDelay = fourPM.getTime() - now.getTime()
    if (nudgeDelay > 0) {
      const futureNonSkipped = blocks.filter((b) => {
        const [h, m] = b.time.split(':').map(Number)
        const bt = new Date(`${today}T${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:00`)
        return !b.skipped && bt > fourPM
      })
      if (futureNonSkipped.length > 0) {
        const id = window.setTimeout(() => {
          new Notification('Review your plan — mark skipped blocks so they don\'t count as completed.')
        }, nudgeDelay)
        timeoutIds.current.push(id)
      }
    }

    return clearAll
  }, [blocks, isAccepted, clearAll])

  return { clearAll }
}
