import { gql } from '@apollo/client'

const PLAN_FIELDS = gql`
  fragment PlanFields on DayPlan {
    id
    planDate
    status
    blocks {
      id
      time
      duration
      title
      category
      taskId
      routineId
      notes
      skipped
      done
    }
    messages {
      id
      role
      content
      createdAt
    }
    createdAt
    updatedAt
  }
`

export const GET_TODAY_PLAN = gql`
  ${PLAN_FIELDS}
  query GetTodayPlan($date: Date!) {
    dayPlan(date: $date) {
      ...PlanFields
    }
  }
`

export const GET_RECENT_PLANS = gql`
  query GetRecentPlans($limit: Int) {
    recentPlans(limit: $limit) {
      id
      planDate
      status
      blocks {
        id
        time
        duration
        title
        category
        taskId
        routineId
        notes
        skipped
      }
    }
  }
`

export const SEND_PLAN_MESSAGE = gql`
  ${PLAN_FIELDS}
  mutation SendPlanMessage($date: Date!, $message: String!) {
    sendPlanMessage(date: $date, message: $message) {
      ...PlanFields
    }
  }
`

export const ACCEPT_PLAN = gql`
  mutation AcceptPlan($date: Date!) {
    acceptPlan(date: $date) {
      id
      status
      blocks {
        id
        time
        duration
        title
        category
        taskId
        routineId
        notes
        skipped
      }
    }
  }
`

export const SKIP_BLOCK = gql`
  mutation SkipBlock($planId: UUID!, $blockId: String!) {
    skipBlock(planId: $planId, blockId: $blockId) {
      id
      blocks {
        id
        time
        duration
        title
        category
        taskId
        routineId
        notes
        skipped
      }
    }
  }
`

export const UNSKIP_BLOCK = gql`
  mutation UnskipBlock($planId: UUID!, $blockId: String!) {
    unskipBlock(planId: $planId, blockId: $blockId) {
      id
      blocks {
        id
        time
        duration
        title
        category
        taskId
        routineId
        notes
        skipped
        done
      }
    }
  }
`

export const COMPLETE_BLOCK = gql`
  mutation CompleteBlock($planId: UUID!, $blockId: String!) {
    completeBlock(planId: $planId, blockId: $blockId) {
      id
      blocks {
        id
        time
        duration
        title
        category
        taskId
        routineId
        notes
        skipped
        done
      }
    }
  }
`

export const UPDATE_BLOCK = gql`
  mutation UpdateBlock($planId: UUID!, $blockId: String!, $input: UpdateBlockInput!) {
    updateBlock(planId: $planId, blockId: $blockId, input: $input) {
      id
      blocks {
        id
        time
        duration
        title
        category
        taskId
        routineId
        notes
        skipped
      }
    }
  }
`

export const GET_CALENDAR_EVENTS_TODAY = gql`
  query CalendarEventsToday($date: Date!) {
    calendarEvents(date: $date) {
      events {
        title
        startTime
        duration
        allDay
      }
      version
      connected
    }
  }
`

export const RESOLVE_SKIPPED_BLOCK = gql`
  mutation ResolveSkippedBlock($planId: UUID!, $blockId: String!, $intentional: Boolean!) {
    resolveSkippedBlock(planId: $planId, blockId: $blockId, intentional: $intentional)
  }
`
