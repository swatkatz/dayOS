import { useState, useMemo } from 'react'
import { useQuery, useMutation } from '@apollo/client/react'
import { GET_CONTEXT_ENTRIES, UPSERT_CONTEXT, TOGGLE_CONTEXT, DELETE_CONTEXT } from '../graphql/manage'

interface ContextEntry {
  id: string
  category: string
  key: string
  value: string
  isActive: boolean
  createdAt: string
}

const CATEGORY_ORDER = ['LIFE', 'CONSTRAINTS', 'EQUIPMENT', 'PREFERENCES', 'CUSTOM']
const CATEGORY_LABELS: Record<string, string> = {
  LIFE: 'Life',
  CONSTRAINTS: 'Constraints',
  EQUIPMENT: 'Equipment',
  PREFERENCES: 'Preferences',
  CUSTOM: 'Custom',
}

function relativeTime(dateStr: string): string {
  const d = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))
  if (diffDays === 0) return 'today'
  if (diffDays === 1) return 'yesterday'
  return `${diffDays} days ago`
}

export default function ContextPage() {
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const [addingCategory, setAddingCategory] = useState<string | null>(null)
  const [newKey, setNewKey] = useState('')
  const [newValue, setNewValue] = useState('')

  const { data, loading } = useQuery<{ contextEntries: ContextEntry[] }>(GET_CONTEXT_ENTRIES)
  const [upsertContext] = useMutation(UPSERT_CONTEXT, {
    refetchQueries: [{ query: GET_CONTEXT_ENTRIES }],
  })
  const [toggleContext] = useMutation(TOGGLE_CONTEXT, {
    refetchQueries: [{ query: GET_CONTEXT_ENTRIES }],
  })
  const [deleteContext] = useMutation(DELETE_CONTEXT, {
    refetchQueries: [{ query: GET_CONTEXT_ENTRIES }],
  })

  const entries: ContextEntry[] = data?.contextEntries ?? []

  const grouped = useMemo(() => {
    const map: Record<string, ContextEntry[]> = {}
    for (const entry of entries) {
      if (!map[entry.category]) map[entry.category] = []
      map[entry.category].push(entry)
    }
    return map
  }, [entries])

  // Show categories that have entries + always show CUSTOM
  const visibleCategories = CATEGORY_ORDER.filter(
    (cat) => (grouped[cat] && grouped[cat].length > 0) || cat === 'CUSTOM'
  )

  const handleEditStart = (entry: ContextEntry) => {
    setEditingId(entry.id)
    setEditValue(entry.value)
  }

  const handleEditSave = async (entry: ContextEntry) => {
    setEditingId(null)
    if (editValue.trim() && editValue !== entry.value) {
      await upsertContext({
        variables: { input: { category: entry.category, key: entry.key, value: editValue.trim() } },
      })
    }
  }

  const handleToggle = async (entry: ContextEntry) => {
    await toggleContext({ variables: { id: entry.id, isActive: !entry.isActive } })
  }

  const handleDelete = async (id: string) => {
    if (window.confirm('Delete this context entry?')) {
      await deleteContext({ variables: { id } })
    }
  }

  const handleAdd = async (category: string) => {
    if (!newKey.trim() || !newValue.trim()) return
    await upsertContext({
      variables: { input: { category, key: newKey.trim(), value: newValue.trim() } },
    })
    setAddingCategory(null)
    setNewKey('')
    setNewValue('')
  }

  if (loading) {
    return <div className="flex items-center justify-center h-64 text-text-secondary">Loading...</div>
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-semibold text-text-primary mb-6">Context</h1>

      {visibleCategories.map((cat) => {
        const catEntries = grouped[cat] ?? []
        return (
          <div key={cat} className="mb-8">
            <h2 className="text-sm font-medium text-text-secondary uppercase tracking-wide border-b border-border-default pb-2 mb-3">
              {CATEGORY_LABELS[cat]}
            </h2>

            <div className="space-y-2">
              {catEntries.map((entry) => (
                <div
                  key={entry.id}
                  className={`bg-bg-surface rounded-lg p-3 border border-border-default group ${!entry.isActive ? 'opacity-50' : ''}`}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex-1 min-w-0">
                      <span className="text-text-primary font-medium text-sm">{entry.key}</span>
                      {editingId === entry.id ? (
                        <textarea
                          value={editValue}
                          onChange={(e) => setEditValue(e.target.value)}
                          onBlur={() => handleEditSave(entry)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleEditSave(entry) }
                            if (e.key === 'Escape') setEditingId(null)
                          }}
                          autoFocus
                          rows={2}
                          className="mt-1 w-full bg-bg-surface border border-accent rounded px-2 py-1 text-text-secondary text-sm outline-none resize-none"
                        />
                      ) : (
                        <p
                          onClick={() => handleEditStart(entry)}
                          className="text-text-secondary text-sm mt-0.5 cursor-pointer hover:text-text-primary transition-colors"
                        >
                          {entry.value}
                        </p>
                      )}
                      {!entry.isActive && (
                        <span className="text-xs text-amber-400 mt-1 block">inactive — not sent to planner</span>
                      )}
                      <span className="text-xs text-text-secondary mt-1 block">{relativeTime(entry.createdAt)}</span>
                    </div>

                    <div className="flex items-center gap-2 flex-shrink-0">
                      <button
                        onClick={() => handleToggle(entry)}
                        className={`w-10 h-5 rounded-full relative transition-colors ${entry.isActive ? 'bg-accent' : 'bg-gray-600'}`}
                      >
                        <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${entry.isActive ? 'left-5' : 'left-0.5'}`} />
                      </button>
                      <button
                        onClick={() => handleDelete(entry.id)}
                        className="text-text-secondary hover:text-red-400 opacity-0 group-hover:opacity-100 transition-all text-sm"
                        title="Delete"
                      >
                        🗑
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            {/* Add entry form */}
            {addingCategory === cat ? (
              <div className="mt-3 bg-bg-surface rounded-lg p-3 border border-border-default">
                <div className="grid grid-cols-1 gap-2 mb-2">
                  <input
                    value={newKey}
                    onChange={(e) => setNewKey(e.target.value)}
                    placeholder="Key"
                    autoFocus
                    className="bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary placeholder:text-text-secondary focus:border-accent outline-none text-sm"
                  />
                  <textarea
                    value={newValue}
                    onChange={(e) => setNewValue(e.target.value)}
                    placeholder="Value"
                    rows={2}
                    className="bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary placeholder:text-text-secondary focus:border-accent outline-none text-sm resize-none"
                  />
                </div>
                <div className="flex gap-2 justify-end">
                  <button onClick={() => setAddingCategory(null)} className="px-3 py-1 text-text-secondary hover:text-text-primary text-sm transition-colors">Cancel</button>
                  <button
                    onClick={() => handleAdd(cat)}
                    disabled={!newKey.trim() || !newValue.trim()}
                    className="px-3 py-1 bg-accent text-black rounded text-sm font-medium hover:bg-accent-hover disabled:opacity-40 transition-colors"
                  >
                    Add
                  </button>
                </div>
              </div>
            ) : (
              <button
                onClick={() => { setAddingCategory(cat); setNewKey(''); setNewValue('') }}
                className="mt-2 text-sm text-text-secondary hover:text-accent transition-colors"
              >
                + Add Entry
              </button>
            )}
          </div>
        )
      })}
    </div>
  )
}
