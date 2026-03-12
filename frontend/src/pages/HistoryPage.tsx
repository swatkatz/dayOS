import { useState } from 'react'
import { useQuery } from '@apollo/client/react'
import { GET_RECENT_PLANS_FULL, GET_DAY_PLAN } from '../graphql/manage'
import BlockList from '../components/today/BlockList'

interface PlanSummary {
  id: string
  planDate: string
  status: string
  blocks: Array<{
    id: string
    time: string
    duration: number
    title: string
    category: string
    taskId: string | null
    routineId: string | null
    notes: string | null
    skipped: boolean
  }>
  createdAt: string
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr + 'T00:00:00')
  return d.toLocaleDateString('en-US', {
    weekday: 'long',
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
}

export default function HistoryPage() {
  const [limit, setLimit] = useState(30)
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const { data, loading } = useQuery<{ recentPlans: PlanSummary[] }>(GET_RECENT_PLANS_FULL, {
    variables: { limit },
  })

  const plans: PlanSummary[] = data?.recentPlans ?? []

  if (loading && plans.length === 0) {
    return <div className="flex items-center justify-center h-64 text-text-secondary">Loading...</div>
  }

  if (plans.length === 0) {
    return (
      <div className="max-w-2xl mx-auto">
        <h1 className="text-2xl font-semibold text-text-primary mb-6">History</h1>
        <div className="text-center text-text-secondary py-16">
          No plans yet. Start planning on the Today page.
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-semibold text-text-primary mb-6">History</h1>

      <div className="space-y-2">
        {plans.map((plan) => {
          const skippedCount = plan.blocks.filter((b) => b.skipped).length
          const isExpanded = expandedId === plan.id

          return (
            <div key={plan.id}>
              <button
                onClick={() => setExpandedId(isExpanded ? null : plan.id)}
                className="w-full text-left bg-bg-surface rounded-lg p-4 border border-border-default hover:bg-bg-surface-hover transition-colors"
              >
                <div className="flex items-center justify-between">
                  <div>
                    <span className="text-text-primary font-medium">
                      {formatDate(plan.planDate)}
                    </span>
                    <span className="text-text-secondary text-sm ml-3">
                      {plan.blocks.length} blocks{skippedCount > 0 ? `, ${skippedCount} skipped` : ''}
                    </span>
                  </div>
                  <span className={`text-xs px-2 py-0.5 rounded ${
                    plan.status === 'ACCEPTED'
                      ? 'bg-accent/20 text-accent'
                      : 'bg-bg-surface-hover text-text-secondary'
                  }`}>
                    {plan.status === 'ACCEPTED' ? 'Accepted' : 'Draft'}
                  </span>
                </div>
              </button>

              {isExpanded && (
                <PlanDetail planDate={plan.planDate} blocks={plan.blocks} />
              )}
            </div>
          )
        })}
      </div>

      {plans.length >= limit && (
        <div className="text-center mt-6">
          <button
            onClick={() => setLimit((l) => l + 30)}
            className="px-4 py-2 border border-border-default text-text-secondary rounded hover:text-text-primary hover:border-accent transition-colors"
          >
            Load more
          </button>
        </div>
      )}
    </div>
  )
}

function PlanDetail({ planDate, blocks }: { planDate: string; blocks: PlanSummary['blocks'] }) {
  const [showChat, setShowChat] = useState(false)

  const { data } = useQuery<{ dayPlan: { messages: Array<{ id: string; role: string; content: string }> } | null }>(GET_DAY_PLAN, {
    variables: { date: planDate },
    skip: !showChat,
  })

  const messages = data?.dayPlan?.messages ?? []

  return (
    <div className="mt-1 ml-4 mr-4 mb-4 p-4 bg-bg-surface rounded-lg border border-border-default">
      <BlockList blocks={blocks} readOnly />

      <button
        onClick={() => setShowChat(!showChat)}
        className="mt-4 text-sm text-text-secondary hover:text-accent transition-colors"
      >
        {showChat ? 'Hide chat history' : 'Show chat history'}
      </button>

      {showChat && messages.length > 0 && (
        <div className="mt-3 space-y-2 border-t border-border-default pt-3">
          {messages.map((msg: { id: string; role: string; content: string }) => (
            <div key={msg.id} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div className={`max-w-[80%] rounded-lg px-3 py-2 text-sm ${
                msg.role === 'user' ? 'bg-accent text-black' : 'bg-bg-surface-hover text-text-primary'
              }`}>
                {msg.content}
              </div>
            </div>
          ))}
        </div>
      )}

      {showChat && messages.length === 0 && (
        <p className="mt-3 text-text-secondary text-sm">No chat messages for this plan.</p>
      )}
    </div>
  )
}
