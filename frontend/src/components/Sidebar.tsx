import { Link, useLocation } from 'react-router-dom'

const NAV_ITEMS = [
  { path: '/', label: 'Today', emoji: '\u2600\uFE0F' },
  { path: '/backlog', label: 'Backlog', emoji: '\uD83D\uDCCB' },
  { path: '/routines', label: 'Routines', emoji: '\uD83D\uDD01' },
  { path: '/context', label: 'Context', emoji: '\uD83E\uDDE0' },
  { path: '/history', label: 'History', emoji: '\uD83D\uDCCA' },
]

export default function Sidebar() {
  const { pathname } = useLocation()

  return (
    <>
      {/* Desktop sidebar */}
      <aside className="hidden md:flex flex-col w-48 min-h-screen bg-bg-surface border-r border-border-default">
        <div className="px-4 py-4">
          <span className="text-lg font-semibold text-accent">🗓️ DayOS</span>
        </div>
        <nav className="flex flex-col">
          {NAV_ITEMS.map(({ path, label, emoji }) => {
            const active = pathname === path
            return (
              <Link
                key={path}
                to={path}
                className={`py-2 px-4 border-l-2 transition-colors ${
                  active
                    ? 'border-accent text-accent'
                    : 'border-transparent text-text-secondary hover:text-text-primary hover:bg-bg-surface-hover'
                }`}
              >
                <span className="mr-2">{emoji}</span>{label}
              </Link>
            )
          })}
        </nav>
      </aside>

      {/* Mobile top nav */}
      <nav className="flex md:hidden items-center gap-1 px-2 py-2 bg-bg-surface border-b border-border-default overflow-x-auto">
        <span className="text-lg font-semibold text-accent px-2 mr-2">🗓️ DayOS</span>
        {NAV_ITEMS.map(({ path, label, emoji }) => {
          const active = pathname === path
          return (
            <Link
              key={path}
              to={path}
              className={`px-3 py-1 rounded text-sm whitespace-nowrap ${
                active
                  ? 'text-accent bg-bg-surface-hover'
                  : 'text-text-secondary hover:text-text-primary'
              }`}
            >
              <span className="mr-1">{emoji}</span>{label}
            </Link>
          )
        })}
      </nav>
    </>
  )
}
