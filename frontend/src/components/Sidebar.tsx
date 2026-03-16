import { Link, useLocation } from 'react-router-dom'
import { useClerk, useUser } from '@clerk/clerk-react'

const NAV_ITEMS = [
  { path: '/', label: 'Today', emoji: '☀️' },
  { path: '/backlog', label: 'Backlog', emoji: '📋' },
  { path: '/routines', label: 'Routines', emoji: '🔁' },
  { path: '/context', label: 'Context', emoji: '🧠' },
  { path: '/history', label: 'History', emoji: '📊' },
]

export default function Sidebar() {
  const { pathname } = useLocation()
  const { signOut } = useClerk()
  const { user } = useUser()

  return (
    <>
      {/* Desktop sidebar */}
      <aside className="hidden md:flex flex-col w-52 min-h-screen bg-bg-surface border-r border-border-default">
        <div className="px-5 py-5">
          <span className="text-lg font-semibold text-accent tracking-tight">🗓️ DayOS</span>
        </div>
        <nav className="flex flex-col gap-0.5 px-2 flex-1">
          {NAV_ITEMS.map(({ path, label, emoji }) => {
            const active = pathname === path
            return (
              <Link
                key={path}
                to={path}
                className={`py-2.5 px-3 rounded-lg transition-all text-sm font-medium ${
                  active
                    ? 'bg-accent-dim text-accent'
                    : 'text-text-secondary hover:text-text-primary hover:bg-bg-surface-hover'
                }`}
              >
                <span className="mr-2.5">{emoji}</span>{label}
              </Link>
            )
          })}
        </nav>
        <div className="px-3 py-4 border-t border-border-default">
          {user && (
            <div className="text-xs text-text-secondary truncate mb-2 px-2">
              {user.primaryEmailAddress?.emailAddress}
            </div>
          )}
          <button
            onClick={() => signOut()}
            className="w-full py-2 px-3 rounded-lg text-sm text-text-secondary hover:text-text-primary hover:bg-bg-surface-hover transition-all text-left"
          >
            <span className="mr-2.5">🚪</span>Sign out
          </button>
        </div>
      </aside>

      {/* Mobile bottom nav — fixed to bottom for thumb reach */}
      <nav className="fixed bottom-0 left-0 right-0 z-50 flex md:hidden items-stretch justify-around bg-bg-surface/95 backdrop-blur-lg border-t border-border-default safe-bottom">
        {NAV_ITEMS.map(({ path, label, emoji }) => {
          const active = pathname === path
          return (
            <Link
              key={path}
              to={path}
              className={`flex flex-col items-center gap-0.5 py-2.5 px-1 min-w-[56px] transition-colors ${
                active
                  ? 'text-accent'
                  : 'text-text-secondary active:text-text-primary'
              }`}
            >
              <span className="text-lg leading-none">{emoji}</span>
              <span className="text-[10px] font-medium leading-none">{label}</span>
            </Link>
          )
        })}
        <button
          onClick={() => signOut()}
          className="flex flex-col items-center gap-0.5 py-2.5 px-1 min-w-[56px] transition-colors text-text-secondary active:text-text-primary"
        >
          <span className="text-lg leading-none">🚪</span>
          <span className="text-[10px] font-medium leading-none">Sign out</span>
        </button>
      </nav>
    </>
  )
}
