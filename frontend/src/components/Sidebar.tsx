import { Link, useLocation } from 'react-router-dom'

const NAV_ITEMS = [
  { path: '/', label: 'Today', emoji: '☀️' },
  { path: '/backlog', label: 'Backlog', emoji: '📋' },
  { path: '/routines', label: 'Routines', emoji: '🔁' },
  { path: '/context', label: 'Context', emoji: '🧠' },
  { path: '/history', label: 'History', emoji: '📊' },
]

export default function Sidebar() {
  const { pathname } = useLocation()

  return (
    <>
      {/* Desktop sidebar */}
      <aside className="hidden md:flex flex-col w-52 min-h-screen bg-bg-surface border-r border-border-default">
        <div className="px-5 py-5">
          <span className="text-lg font-semibold text-accent tracking-tight">🗓️ DayOS</span>
        </div>
        <nav className="flex flex-col gap-0.5 px-2">
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
      </nav>
    </>
  )
}
