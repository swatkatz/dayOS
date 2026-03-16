import { useState, useRef, useEffect, useCallback } from 'react'

interface Message {
  id: string
  role: string
  content: string
  createdAt: string
}

interface Props {
  messages: Message[]
  onSend: (message: string) => void
  loading: boolean
  error?: string | null
  isFirstMessage: boolean
}

function formatAssistantContent(content: string): string {
  const trimmed = content.trim()
  // Strip code fences
  const fenced = trimmed.match(/```(?:json)?\s*\n?([\s\S]*?)```/)
  const inner = fenced ? fenced[1].trim() : trimmed
  try {
    const parsed = JSON.parse(inner)
    if (Array.isArray(parsed) && parsed.length > 0 && parsed[0].time && parsed[0].title) {
      return `Here's your plan with ${parsed.length} blocks. Check the preview →`
    }
  } catch {
    // Not JSON — return as-is
  }
  return content
}

export default function ChatPanel({ messages, onSend, loading, error, isFirstMessage }: Props) {
  const [input, setInput] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  const autoResize = useCallback(() => {
    const el = inputRef.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = Math.min(el.scrollHeight, 150) + 'px'
  }, [])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, error])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = input.trim()
    if (!trimmed || loading) return
    onSend(trimmed)
    setInput('')
    // Reset textarea height
    if (inputRef.current) {
      inputRef.current.style.height = 'auto'
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit(e)
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-y-auto p-3 md:p-4 space-y-3">
        {/* Empty state */}
        {messages.length === 0 && !loading && (
          <div className="flex flex-col items-center justify-center h-full text-center px-4">
            <div className="text-4xl mb-4">☀️</div>
            <h2 className="text-lg font-semibold text-text-primary mb-2">Plan your day</h2>
            <p className="text-text-secondary text-sm max-w-xs leading-relaxed">
              Tell me about your priorities and I'll create a schedule. Mention meetings, deadlines, or energy levels.
            </p>
          </div>
        )}

        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
          >
            <div
              className={`max-w-[85%] md:max-w-[80%] rounded-2xl px-4 py-2.5 text-sm leading-relaxed ${
                msg.role === 'user'
                  ? 'bg-accent text-black rounded-br-md'
                  : 'bg-bg-surface text-text-primary rounded-bl-md'
              }`}
            >
              {msg.role === 'assistant' ? formatAssistantContent(msg.content) : msg.content}
            </div>
          </div>
        ))}

        {loading && (
          <div className="flex justify-start">
            <div className="bg-bg-surface rounded-2xl rounded-bl-md px-4 py-3 text-sm text-text-secondary">
              <span className="inline-flex gap-1">
                <span className="w-1.5 h-1.5 bg-text-secondary rounded-full animate-bounce [animation-delay:0ms]" />
                <span className="w-1.5 h-1.5 bg-text-secondary rounded-full animate-bounce [animation-delay:150ms]" />
                <span className="w-1.5 h-1.5 bg-text-secondary rounded-full animate-bounce [animation-delay:300ms]" />
              </span>
            </div>
          </div>
        )}

        {error && (
          <div className="flex justify-start">
            <div className="bg-red-900/20 border border-red-500/20 rounded-2xl px-4 py-2.5 text-sm text-red-400">
              {error}
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      <form onSubmit={handleSubmit} className="p-3 md:p-4 border-t border-border-default bg-bg-primary/50 backdrop-blur-sm">
        <div className="flex gap-2 items-end">
          <textarea
            ref={inputRef}
            value={input}
            onChange={(e) => { setInput(e.target.value); autoResize() }}
            onKeyDown={handleKeyDown}
            disabled={loading}
            rows={1}
            placeholder={isFirstMessage ? 'Describe your day...' : 'Adjust the plan...'}
            className="flex-1 bg-bg-surface border border-border-default rounded-xl px-4 py-2.5 text-text-primary text-[15px] placeholder:text-text-secondary focus:border-accent focus:ring-1 focus:ring-accent outline-none disabled:opacity-50 resize-none overflow-hidden"
          />
          <button
            type="submit"
            disabled={loading || !input.trim()}
            className="px-4 py-2.5 bg-accent text-black rounded-xl font-medium hover:bg-accent-hover active:scale-95 disabled:opacity-40 disabled:cursor-not-allowed transition-all flex-shrink-0"
          >
            Send
          </button>
        </div>
      </form>
    </div>
  )
}
