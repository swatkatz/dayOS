import { useState, useMemo, useEffect, useCallback } from 'react'
import { useQuery, useMutation } from '@apollo/client/react'
import { GET_TODAY_PLAN, GET_RECENT_PLANS, SEND_PLAN_MESSAGE, ACCEPT_PLAN, SKIP_BLOCK, UNSKIP_BLOCK, COMPLETE_BLOCK, UPDATE_BLOCK, GET_CALENDAR_EVENTS_TODAY } from '../graphql/today'
import { useNotifications } from '../hooks/useNotifications'
import SkippedTasksReview from '../components/today/SkippedTasksReview'
import ChatPanel from '../components/today/ChatPanel'
import PlanPreview from '../components/today/PlanPreview'
import AcceptedPlanView from '../components/today/AcceptedPlanView'
import ReplanBanner from '../components/today/ReplanBanner'

interface Block {
  id: string
  time: string
  duration: number
  title: string
  category: string
  taskId: string | null
  routineId: string | null
  notes: string | null
  skipped: boolean
  done: boolean
}

interface Message {
  id: string
  role: string
  content: string
  createdAt: string
}

interface DayPlan {
  id: string
  planDate: string
  status: string
  blocks: Block[]
  messages: Message[]
  createdAt: string
  updatedAt: string
}

interface GetTodayPlanData {
  dayPlan: DayPlan | null
}

interface RecentPlanSummary {
  id: string
  planDate: string
  status: string
  blocks: Block[]
}

interface GetRecentPlansData {
  recentPlans: RecentPlanSummary[]
}

interface SendPlanMessageData {
  sendPlanMessage: DayPlan
}

interface AcceptPlanData {
  acceptPlan: DayPlan
}

interface BlockMutationData {
  skipBlock?: DayPlan
  unskipBlock?: DayPlan
  completeBlock?: DayPlan
  updateBlock?: DayPlan
}

function formatDate(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function todayDate(): string {
  return formatDate(new Date())
}

function tomorrowDate(): string {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  return formatDate(d)
}

export default function TodayPage() {
  const [selectedDay, setSelectedDay] = useState<'today' | 'tomorrow'>('today')
  const today = useMemo(todayDate, [])
  const tomorrow = useMemo(tomorrowDate, [])
  const date = selectedDay === 'today' ? today : tomorrow
  const isFuture = selectedDay === 'tomorrow'

  const [showReview, setShowReview] = useState(true)
  const [replanning, setReplanning] = useState(false)
  const [chatError, setChatError] = useState<string | null>(null)
  const [pendingMessage, setPendingMessage] = useState<string | null>(null)

  const { data: planData, loading: planLoading } = useQuery<GetTodayPlanData>(GET_TODAY_PLAN, {
    variables: { date },
  })

  const { data: recentData } = useQuery<GetRecentPlansData>(GET_RECENT_PLANS, {
    variables: { limit: 1 },
  })

  // Calendar polling for replan detection
  const POLL_INTERVAL = 15 * 60 * 1000 // 15 minutes
  const [planCalendarVersion, setPlanCalendarVersion] = useState<string | null>(null)
  const [showReplanBanner, setShowReplanBanner] = useState(false)

  const { data: calendarData, refetch: refetchCalendar } = useQuery<{
    calendarEvents: { version: string; connected: boolean }
  }>(GET_CALENDAR_EVENTS_TODAY, {
    variables: { date },
    pollInterval: POLL_INTERVAL,
    fetchPolicy: 'network-only',
  })

  // Refetch calendar on tab focus
  const handleVisibilityChange = useCallback(() => {
    if (!document.hidden) {
      refetchCalendar()
    }
  }, [refetchCalendar])

  useEffect(() => {
    document.addEventListener('visibilitychange', handleVisibilityChange)
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange)
  }, [handleVisibilityChange])

  const [sendMessage, { loading: sending }] = useMutation<SendPlanMessageData>(SEND_PLAN_MESSAGE)
  const [acceptPlan, { loading: accepting }] = useMutation<AcceptPlanData>(ACCEPT_PLAN)
  const [skipBlock] = useMutation<BlockMutationData>(SKIP_BLOCK)
  const [unskipBlock] = useMutation<BlockMutationData>(UNSKIP_BLOCK)
  const [completeBlock] = useMutation<BlockMutationData>(COMPLETE_BLOCK)
  const [updateBlock] = useMutation<BlockMutationData>(UPDATE_BLOCK)

  const plan = planData?.dayPlan
  const blocks = plan?.blocks ?? []
  const messages = plan?.messages ?? []
  const isAccepted = plan?.status === 'ACCEPTED'

  useNotifications(blocks, isAccepted && !replanning, !isFuture)

  // Track calendar version for replan detection (only for today's plan)
  useEffect(() => {
    if (isFuture) return

    const calVersion = calendarData?.calendarEvents?.version
    if (!calVersion || !calendarData?.calendarEvents?.connected) return

    // Store version when plan is first accepted
    if (isAccepted && planCalendarVersion === null) {
      setPlanCalendarVersion(calVersion)
    }

    // Detect changes on accepted plans
    if (isAccepted && planCalendarVersion !== null && calVersion !== planCalendarVersion) {
      setShowReplanBanner(true)
    }
  }, [calendarData, isAccepted, planCalendarVersion, isFuture])

  // Check carry-over: skipped blocks from most recent past plan
  const pastPlan = useMemo(() => {
    const plans = recentData?.recentPlans ?? []
    return plans.find((p) => p.planDate < date) ?? null
  }, [recentData, date])

  const skippedBlocks = useMemo(() => {
    if (!pastPlan) return []
    return pastPlan.blocks.filter((b) => b.skipped && b.taskId)
  }, [pastPlan])

  const needsReview = !isFuture && showReview && skippedBlocks.length > 0

  // Handlers
  const handleSendMessage = async (message: string) => {
    setChatError(null)
    setPendingMessage(message)
    try {
      await sendMessage({
        variables: { date, message },
        update: (cache, { data }) => {
          if (data?.sendPlanMessage) {
            cache.writeQuery({
              query: GET_TODAY_PLAN,
              variables: { date },
              data: { dayPlan: data.sendPlanMessage },
            })
          }
        },
      })
      setPendingMessage(null)
    } catch (err) {
      setChatError(err instanceof Error ? err.message : 'Failed to send message')
      setPendingMessage(null)
    }
  }

  const handleAccept = async (editedBlocks?: Block[]) => {
    try {
      const result = await acceptPlan({
        variables: { date },
        update: (cache, { data }) => {
          if (data?.acceptPlan) {
            const existing = cache.readQuery<GetTodayPlanData>({ query: GET_TODAY_PLAN, variables: { date } })
            cache.writeQuery({
              query: GET_TODAY_PLAN,
              variables: { date },
              data: {
                dayPlan: { ...existing?.dayPlan, ...data.acceptPlan },
              },
            })
          }
        },
      })

      // Patch any blocks that were manually edited in the preview
      if (editedBlocks && result.data?.acceptPlan) {
        const planId = result.data.acceptPlan.id
        const original = new Map(blocks.map((b) => [b.id, b]))
        for (const edited of editedBlocks) {
          const orig = original.get(edited.id)
          if (!orig) {
            // New block added manually — update with all fields
            await updateBlock({ variables: { planId, blockId: edited.id, input: { time: edited.time, duration: edited.duration, notes: edited.notes } } })
          } else if (edited.time !== orig.time || edited.duration !== orig.duration || edited.title !== orig.title || edited.notes !== orig.notes || edited.category !== orig.category) {
            await updateBlock({ variables: { planId, blockId: edited.id, input: { time: edited.time, duration: edited.duration, notes: edited.notes } } })
          }
        }
      }

      setReplanning(false)
    } catch {
      // Accept error
    }
  }

  const handleSkip = async (blockId: string) => {
    try {
      await skipBlock({
        variables: { planId: plan!.id, blockId },
        update: (cache, { data }) => {
          if (data?.skipBlock) {
            const existing = cache.readQuery<GetTodayPlanData>({ query: GET_TODAY_PLAN, variables: { date } })
            cache.writeQuery({
              query: GET_TODAY_PLAN,
              variables: { date },
              data: {
                dayPlan: { ...existing?.dayPlan, blocks: data.skipBlock.blocks },
              },
            })
          }
        },
      })
    } catch {
      // Skip error
    }
  }

  const handleUnskip = async (blockId: string) => {
    try {
      await unskipBlock({
        variables: { planId: plan!.id, blockId },
        update: (cache, { data }) => {
          if (data?.unskipBlock) {
            const existing = cache.readQuery<GetTodayPlanData>({ query: GET_TODAY_PLAN, variables: { date } })
            cache.writeQuery({
              query: GET_TODAY_PLAN,
              variables: { date },
              data: {
                dayPlan: { ...existing?.dayPlan, blocks: data.unskipBlock.blocks },
              },
            })
          }
        },
      })
    } catch {
      // Unskip error
    }
  }

  const handleComplete = async (blockId: string) => {
    try {
      await completeBlock({
        variables: { planId: plan!.id, blockId },
        update: (cache, { data }) => {
          if (data?.completeBlock) {
            const existing = cache.readQuery<GetTodayPlanData>({ query: GET_TODAY_PLAN, variables: { date } })
            cache.writeQuery({
              query: GET_TODAY_PLAN,
              variables: { date },
              data: {
                dayPlan: { ...existing?.dayPlan, blocks: data.completeBlock.blocks },
              },
            })
          }
        },
      })
    } catch {
      // Complete error
    }
  }

  const handleUpdateDuration = async (blockId: string, duration: number) => {
    try {
      await updateBlock({
        variables: { planId: plan!.id, blockId, input: { duration } },
        update: (cache, { data }) => {
          if (data?.updateBlock) {
            const existing = cache.readQuery<GetTodayPlanData>({ query: GET_TODAY_PLAN, variables: { date } })
            cache.writeQuery({
              query: GET_TODAY_PLAN,
              variables: { date },
              data: {
                dayPlan: { ...existing?.dayPlan, blocks: data.updateBlock.blocks },
              },
            })
          }
        },
      })
    } catch {
      // Update error
    }
  }

  // Mobile tab state for draft/replan view
  const [mobileTab, setMobileTab] = useState<'chat' | 'plan'>('chat')

  // Day switch handler — resets transient state
  const handleDaySwitch = (day: 'today' | 'tomorrow') => {
    setSelectedDay(day)
    setReplanning(false)
    setShowReplanBanner(false)
    setPlanCalendarVersion(null)
    setMobileTab('chat')
    setChatError(null)
    setPendingMessage(null)
    setShowReview(true)
  }

  const DayToggle = (
    <div className="flex gap-1 p-1 bg-bg-surface rounded-xl w-fit mx-auto">
      {(['today', 'tomorrow'] as const).map((day) => (
        <button
          key={day}
          onClick={() => handleDaySwitch(day)}
          className={`px-4 py-1.5 rounded-lg text-sm font-medium transition-colors ${
            selectedDay === day
              ? 'bg-accent text-[#0f0f11]'
              : 'text-text-secondary hover:text-text-primary'
          }`}
        >
          {day === 'today' ? 'Today' : 'Tomorrow'}
        </button>
      ))}
    </div>
  )

  if (planLoading) {
    return (
      <div className="flex flex-col gap-3 max-w-2xl mx-auto pt-4">
        <div className="flex justify-center mb-2">{DayToggle}</div>
        <div className="skeleton h-6 w-40" />
        <div className="skeleton h-20 w-full" />
        <div className="skeleton h-20 w-full" />
        <div className="skeleton h-20 w-full" />
      </div>
    )
  }

  // Carry-over review gate
  if (needsReview) {
    return (
      <SkippedTasksReview
        planId={pastPlan!.id}
        blocks={skippedBlocks}
        onDone={() => setShowReview(false)}
      />
    )
  }

  const handleCalendarReplan = async () => {
    setShowReplanBanner(false)
    setReplanning(true)
    try {
      await sendMessage({
        variables: { date, message: 'Calendar events have changed. Replan from now.' },
        update: (cache, { data }) => {
          if (data?.sendPlanMessage) {
            cache.writeQuery({
              query: GET_TODAY_PLAN,
              variables: { date },
              data: { dayPlan: data.sendPlanMessage },
            })
          }
        },
      })
    } catch {
      // Error handled by chat panel
    }
  }

  const handleDismissReplanBanner = () => {
    setShowReplanBanner(false)
    const calVersion = calendarData?.calendarEvents?.version
    if (calVersion) {
      setPlanCalendarVersion(calVersion)
    }
  }

  // Accepted plan view (not replanning)
  if (isAccepted && !replanning) {
    return (
      <div>
        <div className="flex justify-center pt-4 mb-2">{DayToggle}</div>
        {showReplanBanner && (
          <div className="px-4">
            <ReplanBanner onReplan={handleCalendarReplan} onDismiss={handleDismissReplanBanner} />
          </div>
        )}
        <AcceptedPlanView
          blocks={blocks}
          onSkip={handleSkip}
          onUnskip={handleUnskip}
          onComplete={handleComplete}
          onUpdateDuration={handleUpdateDuration}
          onReplan={() => setReplanning(true)}
          readOnly={isFuture}
        />
      </div>
    )
  }

  // Draft / new plan / replanning — show chat + preview split
  const displayMessages = pendingMessage
    ? [...messages, { id: '__pending__', role: 'user', content: pendingMessage, createdAt: new Date().toISOString() }]
    : messages
  const isFirstMessage = messages.length === 0

  return (
    <div className="flex flex-col h-[calc(100vh-4rem)]">
      {/* Day toggle */}
      <div className="flex justify-center pt-3 pb-2 border-b border-border-default bg-bg-primary shrink-0">
        {DayToggle}
      </div>

      <div className="flex flex-col md:flex-row flex-1 min-h-0 gap-0">
        {/* Mobile tab switcher */}
        <div className="flex md:hidden border-b border-border-default">
          <button
            onClick={() => setMobileTab('chat')}
            className={`flex-1 py-3 text-sm font-medium transition-colors ${
              mobileTab === 'chat'
                ? 'text-accent border-b-2 border-accent'
                : 'text-text-secondary'
            }`}
          >
            Chat
          </button>
          <button
            onClick={() => setMobileTab('plan')}
            className={`flex-1 py-3 text-sm font-medium transition-colors relative ${
              mobileTab === 'plan'
                ? 'text-accent border-b-2 border-accent'
                : 'text-text-secondary'
            }`}
          >
            Plan
            {blocks.length > 0 && mobileTab !== 'plan' && (
              <span className="absolute top-2 right-[calc(50%-24px)] w-2 h-2 rounded-full bg-accent" />
            )}
          </button>
        </div>

        {/* Desktop: side-by-side. Mobile: tab content */}
        <div className={`md:w-1/2 md:h-full border-r border-border-default ${mobileTab === 'chat' ? 'flex-1' : 'hidden'} md:block`}>
          <ChatPanel
            messages={displayMessages}
            onSend={handleSendMessage}
            loading={sending}
            error={chatError}
            isFirstMessage={isFirstMessage}
            isFuture={isFuture}
          />
        </div>
        <div className={`md:w-1/2 md:h-full ${mobileTab === 'plan' ? 'flex-1' : 'hidden'} md:block`}>
          <PlanPreview blocks={blocks} onAccept={handleAccept} accepting={accepting} />
        </div>
      </div>
    </div>
  )
}
