import { useState, type FormEvent } from 'react'
import { setToken } from '../apollo'

export default function AuthGate({ onAuth }: { onAuth: () => void }) {
  const [password, setPassword] = useState('')

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!password.trim()) return
    setToken(password.trim())
    onAuth()
  }

  return (
    <div className="flex items-center justify-center min-h-screen bg-bg-primary px-4">
      <form onSubmit={handleSubmit} className="flex flex-col gap-5 w-full max-w-xs">
        <div className="text-center">
          <div className="text-4xl mb-3">🗓️</div>
          <h1 className="text-xl font-semibold text-accent tracking-tight">DayOS</h1>
        </div>
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Enter password"
          autoFocus
          className="px-4 py-3 rounded-xl bg-bg-surface border border-border-default text-text-primary placeholder:text-text-secondary focus:outline-none focus:border-accent focus:ring-1 focus:ring-accent text-[15px]"
        />
        <button
          type="submit"
          className="px-4 py-3 rounded-xl bg-accent text-bg-primary font-semibold hover:bg-accent-hover active:scale-[0.98] transition-all"
        >
          Sign in
        </button>
      </form>
    </div>
  )
}
