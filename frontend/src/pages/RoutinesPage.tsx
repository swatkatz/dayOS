import { useState } from 'react'
import { useQuery, useMutation } from '@apollo/client/react'
import { GET_ROUTINES, CREATE_ROUTINE, UPDATE_ROUTINE, DELETE_ROUTINE } from '../graphql/manage'
import { CATEGORY_COLORS } from '../constants'

interface Routine {
  id: string
  title: string
  category: string
  frequency: string
  daysOfWeek: number[] | null
  preferredTimeOfDay: string | null
  preferredDurationMin: number | null
  notes: string | null
  isActive: boolean
}

const CATEGORIES = ['JOB', 'INTERVIEW', 'PROJECT', 'MEAL', 'BABY', 'EXERCISE', 'ADMIN']
const FREQUENCIES = ['DAILY', 'WEEKDAYS', 'WEEKLY', 'CUSTOM']
const TIMES_OF_DAY = ['MORNING', 'MIDDAY', 'AFTERNOON', 'EVENING', 'ANY']
const DAY_NAMES = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

function appliesToday(routine: Routine): boolean {
  const dow = new Date().getDay()
  if (routine.frequency === 'DAILY') return true
  if (routine.frequency === 'WEEKDAYS') return dow >= 1 && dow <= 5
  if ((routine.frequency === 'WEEKLY' || routine.frequency === 'CUSTOM') && routine.daysOfWeek) {
    return routine.daysOfWeek.includes(dow)
  }
  return false
}

function frequencyLabel(routine: Routine): string {
  if (routine.frequency === 'DAILY') return 'Daily'
  if (routine.frequency === 'WEEKDAYS') return 'Weekdays'
  if (routine.frequency === 'WEEKLY' || routine.frequency === 'CUSTOM') {
    if (routine.daysOfWeek && routine.daysOfWeek.length > 0) {
      const days = routine.daysOfWeek.map((d) => DAY_NAMES[d]).join(', ')
      return routine.frequency === 'WEEKLY' ? `Weekly (${days})` : `Custom (${days})`
    }
    return routine.frequency.charAt(0) + routine.frequency.slice(1).toLowerCase()
  }
  return routine.frequency
}

const EMPTY_FORM = {
  title: '',
  category: 'EXERCISE',
  frequency: 'DAILY',
  daysOfWeek: [] as number[],
  preferredTimeOfDay: 'ANY',
  preferredDurationMin: 45,
  notes: '',
}

export default function RoutinesPage() {
  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [form, setForm] = useState(EMPTY_FORM)
  const [validationError, setValidationError] = useState<string | null>(null)

  const { data, loading } = useQuery<{ routines: Routine[] }>(GET_ROUTINES, {
    variables: { activeOnly: false },
  })

  const [createRoutine, { loading: creating }] = useMutation(CREATE_ROUTINE, {
    refetchQueries: [{ query: GET_ROUTINES, variables: { activeOnly: false } }],
  })
  const [updateRoutine] = useMutation(UPDATE_ROUTINE, {
    refetchQueries: [{ query: GET_ROUTINES, variables: { activeOnly: false } }],
  })
  const [deleteRoutine] = useMutation(DELETE_ROUTINE, {
    refetchQueries: [{ query: GET_ROUTINES, variables: { activeOnly: false } }],
  })

  const routines: Routine[] = data?.routines ?? []

  const openAdd = () => {
    setForm(EMPTY_FORM)
    setEditingId(null)
    setValidationError(null)
    setShowForm(true)
  }

  const openEdit = (r: Routine) => {
    setForm({
      title: r.title,
      category: r.category,
      frequency: r.frequency,
      daysOfWeek: r.daysOfWeek ?? [],
      preferredTimeOfDay: r.preferredTimeOfDay ?? 'ANY',
      preferredDurationMin: r.preferredDurationMin ?? 45,
      notes: r.notes ?? '',
    })
    setEditingId(r.id)
    setValidationError(null)
    setShowForm(true)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.title.trim()) return

    if ((form.frequency === 'WEEKLY' || form.frequency === 'CUSTOM') && form.daysOfWeek.length === 0) {
      setValidationError('Select at least one day')
      return
    }
    setValidationError(null)

    const input = {
      title: form.title.trim(),
      category: form.category,
      frequency: form.frequency,
      daysOfWeek: (form.frequency === 'WEEKLY' || form.frequency === 'CUSTOM') ? form.daysOfWeek : null,
      preferredTimeOfDay: form.preferredTimeOfDay,
      preferredDurationMin: form.preferredDurationMin,
      notes: form.notes.trim() || null,
    }

    if (editingId) {
      await updateRoutine({ variables: { id: editingId, input } })
    } else {
      await createRoutine({ variables: { input } })
    }

    setShowForm(false)
    setEditingId(null)
  }

  const handleDelete = async (id: string) => {
    if (window.confirm('Delete this routine?')) {
      await deleteRoutine({ variables: { id } })
    }
  }

  const handleToggle = async (r: Routine) => {
    await updateRoutine({ variables: { id: r.id, input: { isActive: !r.isActive } } })
  }

  const toggleDay = (day: number) => {
    setForm((prev) => ({
      ...prev,
      daysOfWeek: prev.daysOfWeek.includes(day)
        ? prev.daysOfWeek.filter((d) => d !== day)
        : [...prev.daysOfWeek, day],
    }))
  }

  if (loading) {
    return <div className="flex items-center justify-center h-64 text-text-secondary">Loading...</div>
  }

  return (
    <div className="max-w-2xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-text-primary">Routines</h1>
        <button
          onClick={openAdd}
          className="px-4 py-2 bg-accent text-black rounded font-medium hover:bg-accent-hover transition-colors"
        >
          Add Routine
        </button>
      </div>

      {/* Form */}
      {showForm && (
        <form onSubmit={handleSubmit} className="bg-bg-surface rounded-lg p-4 border border-border-default mb-6">
          <div className="grid grid-cols-2 gap-3 mb-3">
            <input
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              placeholder="Routine title"
              autoFocus
              className="col-span-2 bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary placeholder:text-text-secondary focus:border-accent outline-none"
            />
            <select
              value={form.category}
              onChange={(e) => setForm({ ...form, category: e.target.value })}
              className="bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary focus:border-accent outline-none"
            >
              {CATEGORIES.map((c) => <option key={c} value={c}>{c.toLowerCase()}</option>)}
            </select>
            <select
              value={form.frequency}
              onChange={(e) => setForm({ ...form, frequency: e.target.value })}
              className="bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary focus:border-accent outline-none"
            >
              {FREQUENCIES.map((f) => <option key={f} value={f}>{f.toLowerCase()}</option>)}
            </select>

            {(form.frequency === 'WEEKLY' || form.frequency === 'CUSTOM') && (
              <div className="col-span-2 flex gap-2 flex-wrap">
                {DAY_NAMES.map((name, i) => (
                  <button
                    key={i}
                    type="button"
                    onClick={() => toggleDay(i)}
                    className={`px-3 py-1 rounded text-sm transition-colors ${
                      form.daysOfWeek.includes(i)
                        ? 'bg-accent text-black'
                        : 'bg-bg-surface-hover text-text-secondary hover:text-text-primary'
                    }`}
                  >
                    {name}
                  </button>
                ))}
                {validationError && (
                  <span className="text-red-400 text-sm w-full">{validationError}</span>
                )}
              </div>
            )}

            <select
              value={form.preferredTimeOfDay}
              onChange={(e) => setForm({ ...form, preferredTimeOfDay: e.target.value })}
              className="bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary focus:border-accent outline-none"
            >
              {TIMES_OF_DAY.map((t) => <option key={t} value={t}>{t.toLowerCase()}</option>)}
            </select>
            <div className="flex items-center gap-2">
              <input
                type="number"
                value={form.preferredDurationMin}
                onChange={(e) => setForm({ ...form, preferredDurationMin: parseInt(e.target.value) || 45 })}
                min={1}
                className="w-20 bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary focus:border-accent outline-none"
              />
              <span className="text-text-secondary text-sm">min</span>
            </div>
            <textarea
              value={form.notes}
              onChange={(e) => setForm({ ...form, notes: e.target.value })}
              placeholder="Notes (optional)"
              rows={2}
              className="col-span-2 bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary placeholder:text-text-secondary focus:border-accent outline-none resize-none"
            />
          </div>
          <div className="flex gap-2 justify-end">
            <button type="button" onClick={() => setShowForm(false)} className="px-3 py-1.5 text-text-secondary hover:text-text-primary transition-colors">
              Cancel
            </button>
            <button type="submit" disabled={!form.title.trim() || creating} className="px-4 py-1.5 bg-accent text-black rounded font-medium hover:bg-accent-hover disabled:opacity-40 transition-colors">
              {editingId ? 'Save' : 'Add'}
            </button>
          </div>
        </form>
      )}

      {/* Routines list */}
      {routines.length === 0 ? (
        <div className="text-center text-text-secondary py-16">
          No routines yet. Add one to get started.
        </div>
      ) : (
        <div className="space-y-3">
          {routines.map((r) => {
            const color = CATEGORY_COLORS[r.category] || '#6b7280'
            const today = appliesToday(r)
            return (
              <div
                key={r.id}
                className={`bg-bg-surface rounded-lg p-4 border border-border-default group ${!r.isActive ? 'opacity-50' : ''}`}
                style={{ borderLeft: `4px solid ${color}` }}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      {today && r.isActive && (
                        <span className="w-2 h-2 rounded-full bg-emerald-500 flex-shrink-0" title="Applies today" />
                      )}
                      <span className="text-text-primary font-medium">{r.title}</span>
                      <span className="text-xs px-2 py-0.5 rounded" style={{ color, borderColor: color, border: '1px solid' }}>
                        {r.category.toLowerCase()}
                      </span>
                    </div>
                    <div className="flex items-center gap-3 text-text-secondary text-sm">
                      <span>{frequencyLabel(r)}</span>
                      {r.preferredTimeOfDay && (
                        <span>{r.preferredTimeOfDay.toLowerCase()}</span>
                      )}
                      {r.preferredDurationMin && (
                        <span>{r.preferredDurationMin} min</span>
                      )}
                    </div>
                    {r.notes && (
                      <p className="text-text-secondary text-sm mt-1 truncate">{r.notes}</p>
                    )}
                  </div>

                  <div className="flex items-center gap-3 flex-shrink-0">
                    {/* Toggle */}
                    <button
                      onClick={() => handleToggle(r)}
                      className={`w-10 h-5 rounded-full relative transition-colors ${r.isActive ? 'bg-accent' : 'bg-gray-600'}`}
                    >
                      <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${r.isActive ? 'left-5' : 'left-0.5'}`} />
                    </button>

                    {/* Edit */}
                    <button
                      onClick={() => openEdit(r)}
                      className="text-text-secondary hover:text-text-primary opacity-0 group-hover:opacity-100 transition-all"
                      title="Edit"
                    >
                      ✎
                    </button>

                    {/* Delete */}
                    <button
                      onClick={() => handleDelete(r.id)}
                      className="text-text-secondary hover:text-red-400 opacity-0 group-hover:opacity-100 transition-all"
                      title="Delete"
                    >
                      🗑
                    </button>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
