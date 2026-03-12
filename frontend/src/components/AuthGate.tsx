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
    <div className="flex items-center justify-center min-h-screen bg-bg-primary">
      <form onSubmit={handleSubmit} className="flex flex-col gap-4 w-80">
        <h1 className="text-lg font-semibold text-accent text-center">DayOS</h1>
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Enter password"
          autoFocus
          className="px-4 py-2 rounded bg-bg-surface border border-border-default text-text-primary placeholder:text-text-secondary focus:outline-none focus:border-accent"
        />
        <button
          type="submit"
          className="px-4 py-2 rounded bg-accent text-bg-primary font-medium hover:bg-accent-hover"
        >
          Sign in
        </button>
      </form>
    </div>
  )
}
