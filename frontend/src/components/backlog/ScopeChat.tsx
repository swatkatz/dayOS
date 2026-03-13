import { useState, useRef, useEffect, useCallback } from 'react'
import { useMutation } from '@apollo/client/react'
import { START_TASK_CONVERSATION, SEND_TASK_MESSAGE, CONFIRM_TASK_BREAKDOWN, GET_TASKS } from '../../graphql/backlog'

interface Message {
  id: string
  role: string
  content: string
}

interface Props {
  onClose: () => void
}

interface Proposal {
  parent: {
    title: string
    category: string
    priority: string
    deadline_type?: string | null
    deadline_date?: string | null
    deadline_days?: number | null
  }
  subtasks: Array<{
    title: string
    estimated_minutes: number
    category?: string
    notes?: string
  }>
}

function extractJSON(content: string): string {
  const fenced = content.match(/```(?:json)?\s*\n?([\s\S]*?)```/)
  return fenced ? fenced[1].trim() : content.trim()
}

function tryParseProposal(content: string): { type: 'proposal'; data: Proposal } | { type: 'question'; message: string } | null {
  try {
    const parsed = JSON.parse(extractJSON(content))
    if (parsed.status === 'proposal' && parsed.parent && parsed.subtasks) {
      return { type: 'proposal', data: parsed }
    }
    if (parsed.status === 'question' && parsed.message) {
      return { type: 'question', message: parsed.message }
    }
  } catch {
    // Not JSON — render as plain text
  }
  return null
}

function formatMinutes(mins: number): string {
  if (mins < 60) return `${mins}min`
  const h = Math.floor(mins / 60)
  const m = mins % 60
  return m > 0 ? `${h}hr ${m}min` : `${h}hr`
}

function formatDeadline(proposal: Proposal): string | null {
  const { deadline_type, deadline_date, deadline_days } = proposal.parent
  if (deadline_type === 'hard' && deadline_date) {
    const d = new Date(deadline_date + 'T00:00:00')
    return `due ${d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}`
  }
  if (deadline_type === 'horizon' && deadline_days) {
    return `within ${deadline_days} days`
  }
  return null
}

export default function ScopeChat({ onClose }: Props) {
  const [messages, setMessages] = useState<Message[]>([])
  const [conversationId, setConversationId] = useState<string | null>(null)
  const [input, setInput] = useState('')
  const [proposalData, setProposalData] = useState<Proposal | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  const autoResize = useCallback(() => {
    const el = inputRef.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = Math.min(el.scrollHeight, 150) + 'px'
  }, [])

  const [startConversation, { loading: starting }] = useMutation<{ startTaskConversation: { id: string; status: string; messages: Message[] } }>(START_TASK_CONVERSATION)
  const [sendMessage, { loading: sendingMsg }] = useMutation<{ sendTaskMessage: { id: string; status: string; messages: Message[] } }>(SEND_TASK_MESSAGE)
  const [confirmBreakdown, { loading: confirming }] = useMutation(CONFIRM_TASK_BREAKDOWN, {
    refetchQueries: [{ query: GET_TASKS, variables: { includeCompleted: true } }],
  })

  const loading = starting || sendingMsg

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const handleSend = async () => {
    const trimmed = input.trim()
    if (!trimmed || loading) return
    setInput('')
    if (inputRef.current) inputRef.current.style.height = 'auto'

    // Show user message immediately
    const pendingMsg: Message = { id: '__pending__', role: 'user', content: trimmed }
    setMessages((prev) => [...prev, pendingMsg])

    try {
      if (!conversationId) {
        const { data } = await startConversation({ variables: { message: trimmed } })
        const conv = data?.startTaskConversation
        if (conv) {
          setConversationId(conv.id)
          setMessages(conv.messages)
          checkForProposal(conv.messages)
        }
      } else {
        const { data } = await sendMessage({ variables: { conversationId, message: trimmed } })
        const conv = data?.sendTaskMessage
        if (conv) {
          setMessages(conv.messages)
          checkForProposal(conv.messages)
        }
      }
    } catch {
      // Remove pending message on error so it doesn't stick
      setMessages((prev) => prev.filter((m) => m.id !== '__pending__'))
    }
  }

  const checkForProposal = (msgs: Message[]) => {
    const lastAssistant = [...msgs].reverse().find((m) => m.role === 'assistant')
    if (lastAssistant) {
      const parsed = tryParseProposal(lastAssistant.content)
      if (parsed?.type === 'proposal') {
        setProposalData(parsed.data)
      } else {
        setProposalData(null)
      }
    }
  }

  const [error, setError] = useState<string | null>(null)

  const handleConfirm = async () => {
    if (!conversationId) return
    setError(null)
    try {
      await confirmBreakdown({ variables: { conversationId } })
      onClose()
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to create tasks'
      setError(msg)
    }
  }

  const handleAdjust = () => {
    setInput("I'd like to adjust ")
    setProposalData(null)
  }

  const renderMessage = (msg: Message) => {
    if (msg.role === 'assistant') {
      const parsed = tryParseProposal(msg.content)
      if (parsed?.type === 'question') {
        return <span>{parsed.message}</span>
      }
      if (parsed?.type === 'proposal') {
        const p = parsed.data
        const totalMin = p.subtasks.reduce((sum, s) => sum + s.estimated_minutes, 0)
        const deadline = formatDeadline(p)
        return (
          <div className="space-y-2">
            <div className="font-medium">Proposed Breakdown</div>
            <div className="text-sm">
              <div>Parent: {p.parent.title}</div>
              <div>Category: {p.parent.category} | Priority: {p.parent.priority}</div>
              {deadline && <div>Deadline: {deadline}</div>}
            </div>
            <div className="text-sm mt-2">
              <div className="font-medium mb-1">Subtasks:</div>
              {p.subtasks.map((s, i) => (
                <div key={i} className="ml-2">
                  {i + 1}. {s.title} — {formatMinutes(s.estimated_minutes)}
                </div>
              ))}
            </div>
            <div className="text-xs text-text-secondary mt-1">
              Total: {formatMinutes(totalMin)}
            </div>
          </div>
        )
      }
    }
    return <span>{msg.content}</span>
  }

  return (
    <div className="fixed inset-y-0 right-0 w-full md:w-[28rem] bg-bg-primary border-l border-border-default z-50 flex flex-col shadow-2xl">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-border-default">
        <h3 className="font-medium text-text-primary">Scope with AI</h3>
        <button onClick={onClose} className="text-text-secondary hover:text-text-primary">✕</button>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {messages.length === 0 && (
          <p className="text-text-secondary text-sm text-center mt-8">
            Describe a goal and I'll help break it into actionable tasks.
          </p>
        )}
        {messages.map((msg) => (
          <div key={msg.id} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
            <div className={`max-w-[85%] rounded-lg px-4 py-2 text-sm ${
              msg.role === 'user' ? 'bg-accent text-black' : 'bg-bg-surface text-text-primary'
            }`}>
              {renderMessage(msg)}
            </div>
          </div>
        ))}
        {loading && (
          <div className="flex justify-start">
            <div className="bg-bg-surface rounded-lg px-4 py-2 text-sm text-text-secondary animate-pulse">
              Thinking...
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Proposal actions */}
      {error && (
        <div className="px-4 pt-3 text-sm text-red-400">{error}</div>
      )}
      {proposalData && (
        <div className="p-4 border-t border-border-default flex gap-2">
          <button
            onClick={handleConfirm}
            disabled={confirming}
            className="flex-1 py-2 bg-accent text-black rounded font-medium hover:bg-accent-hover disabled:opacity-40 transition-colors"
          >
            {confirming ? 'Creating...' : 'Create Tasks'}
          </button>
          <button
            onClick={handleAdjust}
            className="px-4 py-2 text-text-secondary hover:text-text-primary border border-border-default rounded transition-colors"
          >
            Adjust...
          </button>
        </div>
      )}

      {/* Input */}
      <form
        onSubmit={(e) => { e.preventDefault(); handleSend() }}
        className="p-4 border-t border-border-default"
      >
        <div className="flex gap-2">
          <textarea
            ref={inputRef}
            value={input}
            onChange={(e) => { setInput(e.target.value); autoResize() }}
            onKeyDown={handleKeyDown}
            disabled={loading}
            rows={1}
            placeholder="Describe a goal..."
            className="flex-1 bg-bg-surface border border-border-default rounded px-3 py-2 text-text-primary placeholder:text-text-secondary focus:border-accent focus:ring-1 focus:ring-accent outline-none disabled:opacity-50 resize-none overflow-hidden"
          />
          <button
            type="submit"
            disabled={loading || !input.trim()}
            className="px-4 py-2 bg-accent text-black rounded font-medium hover:bg-accent-hover disabled:opacity-40 transition-colors"
          >
            Send
          </button>
        </div>
      </form>
    </div>
  )
}
