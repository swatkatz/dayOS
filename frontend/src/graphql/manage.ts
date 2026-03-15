import { gql } from '@apollo/client'

export const GET_ROUTINES = gql`
  query Routines($activeOnly: Boolean) {
    routines(activeOnly: $activeOnly) {
      id
      title
      category
      frequency
      daysOfWeek
      preferredTimeOfDay
      preferredDurationMin
      preferredExactTime
      notes
      isActive
    }
  }
`

export const CREATE_ROUTINE = gql`
  mutation CreateRoutine($input: CreateRoutineInput!) {
    createRoutine(input: $input) {
      id
      title
      category
      frequency
      daysOfWeek
      preferredTimeOfDay
      preferredDurationMin
      preferredExactTime
      notes
      isActive
    }
  }
`

export const UPDATE_ROUTINE = gql`
  mutation UpdateRoutine($id: UUID!, $input: UpdateRoutineInput!) {
    updateRoutine(id: $id, input: $input) {
      id
      title
      category
      frequency
      daysOfWeek
      preferredTimeOfDay
      preferredDurationMin
      preferredExactTime
      notes
      isActive
    }
  }
`

export const DELETE_ROUTINE = gql`
  mutation DeleteRoutine($id: UUID!) {
    deleteRoutine(id: $id)
  }
`

export const GET_CONTEXT_ENTRIES = gql`
  query ContextEntries($category: ContextCategory) {
    contextEntries(category: $category) {
      id
      category
      key
      value
      isActive
      createdAt
    }
  }
`

export const UPSERT_CONTEXT = gql`
  mutation UpsertContext($input: UpsertContextInput!) {
    upsertContext(input: $input) {
      id
      category
      key
      value
      isActive
      createdAt
    }
  }
`

export const TOGGLE_CONTEXT = gql`
  mutation ToggleContext($id: UUID!, $isActive: Boolean!) {
    toggleContext(id: $id, isActive: $isActive) {
      id
      isActive
    }
  }
`

export const DELETE_CONTEXT = gql`
  mutation DeleteContext($id: UUID!) {
    deleteContext(id: $id)
  }
`

export const GET_RECENT_PLANS_FULL = gql`
  query RecentPlansFull($limit: Int) {
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
      createdAt
    }
  }
`

export const GET_CALENDAR_EVENTS = gql`
  query CalendarEvents($date: Date!) {
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

export const GET_GOOGLE_CALENDAR_STATUS = gql`
  query GoogleCalendarStatus {
    googleCalendarStatus {
      connected
      calendarName
    }
  }
`

export const CONNECT_GOOGLE_CALENDAR = gql`
  mutation ConnectGoogleCalendar($code: String!) {
    connectGoogleCalendar(code: $code)
  }
`

export const DISCONNECT_GOOGLE_CALENDAR = gql`
  mutation DisconnectGoogleCalendar {
    disconnectGoogleCalendar
  }
`

export const GET_DAY_PLAN = gql`
  query DayPlan($date: Date!) {
    dayPlan(date: $date) {
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
      messages {
        id
        role
        content
        createdAt
      }
      createdAt
      updatedAt
    }
  }
`
