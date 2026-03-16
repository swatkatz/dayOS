import { useEffect, useRef, useCallback } from 'react'

interface Particle {
  x: number
  y: number
  color: string
  rotation: number
  dx: number
  dy: number
  dr: number
  width: number
  height: number
  opacity: number
  wobbleSpeed: number
  wobbleAmp: number
  wobbleOffset: number
}

const COLORS = [
  '#c5a55a', '#d4b86a', // golds
  '#6366f1', '#818cf8', // indigo
  '#10b981', '#34d399', // emerald
  '#f59e0b', '#fbbf24', // amber
  '#ef4444', '#f87171', // red
  '#0ea5e9', '#38bdf8', // sky
  '#8b5cf6', '#a78bfa', // violet
  '#14b8a6', '#2dd4bf', // teal
]

function createParticles(): Particle[] {
  const particles: Particle[] = []
  const count = 150
  const w = window.innerWidth

  for (let i = 0; i < count; i++) {
    const pw = 8 + Math.random() * 14
    const isStrip = Math.random() > 0.3
    const ph = isStrip ? pw * (2.5 + Math.random() * 2) : pw

    particles.push({
      x: Math.random() * w,
      y: -20 - Math.random() * 400, // stagger spawn above viewport
      color: COLORS[Math.floor(Math.random() * COLORS.length)],
      rotation: Math.random() * 360,
      dx: (Math.random() - 0.5) * 2,
      dy: 3 + Math.random() * 4, // fall speed
      dr: (Math.random() - 0.5) * 6,
      width: pw,
      height: ph,
      opacity: 0.9 + Math.random() * 0.1,
      wobbleSpeed: 0.02 + Math.random() * 0.04,
      wobbleAmp: 30 + Math.random() * 60,
      wobbleOffset: Math.random() * Math.PI * 2,
    })
  }

  return particles
}

interface Props {
  onDone: () => void
}

export default function Confetti({ onDone }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const particlesRef = useRef<Particle[]>(createParticles())
  const frameRef = useRef(0)
  const rafRef = useRef<number>(0)
  const startRef = useRef(performance.now())

  const DURATION_MS = 2000 // total animation time
  const FADE_START_MS = 1400 // when fade-out begins

  const draw = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const w = window.innerWidth
    const h = window.innerHeight
    if (canvas.width !== w) canvas.width = w
    if (canvas.height !== h) canvas.height = h

    const elapsed = performance.now() - startRef.current
    if (elapsed > DURATION_MS) {
      onDone()
      return
    }

    // Global fade for final stretch
    const globalAlpha = elapsed > FADE_START_MS
      ? 1 - (elapsed - FADE_START_MS) / (DURATION_MS - FADE_START_MS)
      : 1

    ctx.clearRect(0, 0, w, h)

    const frame = frameRef.current
    const particles = particlesRef.current

    for (const p of particles) {
      // Wobble side to side like real confetti
      const wobbleX = Math.sin(frame * p.wobbleSpeed + p.wobbleOffset) * p.wobbleAmp * 0.03

      p.x += p.dx + wobbleX
      p.y += p.dy
      p.rotation += p.dr

      // Wrap horizontally so confetti doesn't disappear off sides
      if (p.x < -20) p.x = w + 20
      if (p.x > w + 20) p.x = -20

      // Skip if below screen
      if (p.y > h + 50) continue

      ctx.save()
      ctx.translate(p.x, p.y)
      ctx.rotate((p.rotation * Math.PI) / 180)
      ctx.globalAlpha = p.opacity * globalAlpha
      ctx.fillStyle = p.color

      // Rounded rect confetti piece
      const hw = p.width / 2
      const hh = p.height / 2
      const r = Math.min(hw, hh, 3)
      ctx.beginPath()
      ctx.moveTo(-hw + r, -hh)
      ctx.lineTo(hw - r, -hh)
      ctx.quadraticCurveTo(hw, -hh, hw, -hh + r)
      ctx.lineTo(hw, hh - r)
      ctx.quadraticCurveTo(hw, hh, hw - r, hh)
      ctx.lineTo(-hw + r, hh)
      ctx.quadraticCurveTo(-hw, hh, -hw, hh - r)
      ctx.lineTo(-hw, -hh + r)
      ctx.quadraticCurveTo(-hw, -hh, -hw + r, -hh)
      ctx.fill()

      ctx.restore()
    }

    frameRef.current++
    rafRef.current = requestAnimationFrame(draw)
  }, [onDone])

  useEffect(() => {
    startRef.current = performance.now()
    rafRef.current = requestAnimationFrame(draw)
    return () => cancelAnimationFrame(rafRef.current)
  }, [draw])

  return (
    <canvas
      ref={canvasRef}
      className="fixed inset-0 pointer-events-none z-[100]"
      style={{ width: '100vw', height: '100vh' }}
    />
  )
}
