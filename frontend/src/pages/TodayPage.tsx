import { useState, useMemo } from 'react'
import { useQuery, useMutation } from '@apollo/client/react'
import { GET_TODAY_PLAN, GET_RECENT_PLANS, SEND_PLAN_MESSAGE, ACCEPT_PLAN, SKIP_BLOCK, UPDATE_BLOCK } from '../graphql/today'
import { useNotifications } from '../hooks/useNotifications'
import SkippedTasksReview from '../components/today/SkippedTasksReview'
import ChatPanel from '../components/today/ChatPanel'
import PlanPreview from '../components/today/PlanPreview'
import AcceptedPlanView from '../components/today/AcceptedPlanView'

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
  updateBlock?: DayPlan
}

function todayDate(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

export default function TodayPage() {
  const date = useMemo(todayDate, [])
  const [showReview, setShowReview] = useState(true)
  const [replanning, setReplanning] = useState(false)
  const [chatError, setChatError] = useState<string | null>(null)

  const { data: planData, loading: planLoading } = useQuery<GetTodayPlanData>(GET_TODAY_PLAN, {
    variables: { date },
  })

  const { data: recentData } = useQuery<GetRecentPlansData>(GET_RECENT_PLANS, {
    variables: { limit: 1 },
  })

  const [sendMessage, { loading: sending }] = useMutation<SendPlanMessageData>(SEND_PLAN_MESSAGE)
  const [acceptPlan, { loading: accepting }] = useMutation<AcceptPlanData>(ACCEPT_PLAN)
  const [skipBlock] = useMutation<BlockMutationData>(SKIP_BLOCK)
  const [updateBlock] = useMutation<BlockMutationData>(UPDATE_BLOCK)

  const plan = planData?.dayPlan
  const blocks = plan?.blocks ?? []
  const messages = plan?.messages ?? []
  const isAccepted = plan?.status === 'ACCEPTED'

  useNotifications(blocks, isAccepted && !replanning)

  // Check carry-over: skipped blocks from most recent past plan
  const pastPlan = useMemo(() => {
    const plans = recentData?.recentPlans ?? []
    return plans.find((p) => p.planDate < date) ?? null
  }, [recentData, date])

  const skippedBlocks = useMemo(() => {
    if (!pastPlan) return []
    return pastPlan.blocks.filter((b) => b.skipped && b.taskId)
  }, [pastPlan])

  const needsReview = showReview && skippedBlocks.length > 0

  // Handlers
  const handleSendMessage = async (message: string) => {
    setChatError(null)
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
    } catch (err) {
      setChatError(err instanceof Error ? err.message : 'Failed to send message')
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

  if (planLoading) {
    return (
      <div className="flex items-center justify-center h-64 text-text-secondary">
        Loading...
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

  // Accepted plan view (not replanning)
  if (isAccepted && !replanning) {
    return (
      <AcceptedPlanView
        blocks={blocks}
        onSkip={handleSkip}
        onUpdateDuration={handleUpdateDuration}
        onReplan={() => setReplanning(true)}
      />
    )
  }

  // Draft / new plan / replanning — show chat + preview split
  const isFirstMessage = messages.length === 0

  return (
    <div className="flex flex-col md:flex-row h-[calc(100vh-4rem)] gap-0">
      <div className="md:w-1/2 h-1/2 md:h-full border-r border-border-default">
        <ChatPanel
          messages={messages}
          onSend={handleSendMessage}
          loading={sending}
          error={chatError}
          isFirstMessage={isFirstMessage}
        />
      </div>
      <div className="md:w-1/2 h-1/2 md:h-full">
        <PlanPreview blocks={blocks} onAccept={handleAccept} accepting={accepting} />
      </div>
    </div>
  )
}
