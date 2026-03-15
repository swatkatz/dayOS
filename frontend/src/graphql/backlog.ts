import { gql } from '@apollo/client'

export const GET_TASKS = gql`
  query Tasks($category: Category, $includeCompleted: Boolean) {
    tasks(category: $category, includeCompleted: $includeCompleted) {
      id
      title
      category
      priority
      parentId
      estimatedMinutes
      actualMinutes
      deadlineType
      deadlineDate
      deadlineDays
      notes
      isRoutine
      timesDeferred
      isCompleted
      completedAt
      createdAt
      subtasks {
        id
        title
        category
        priority
        estimatedMinutes
        actualMinutes
        deadlineType
        deadlineDate
        deadlineDays
        notes
        isCompleted
        completedAt
        timesDeferred
      }
    }
  }
`

export const GET_TASK_CONVERSATION = gql`
  query TaskConversation($id: UUID!) {
    taskConversation(id: $id) {
      id
      parentTaskId
      status
      messages {
        id
        role
        content
        createdAt
      }
    }
  }
`

export const CREATE_TASK = gql`
  mutation CreateTask($input: CreateTaskInput!) {
    createTask(input: $input) {
      id
      title
      category
      priority
      estimatedMinutes
      deadlineType
      deadlineDate
      deadlineDays
    }
  }
`

export const UPDATE_TASK = gql`
  mutation UpdateTask($id: UUID!, $input: UpdateTaskInput!) {
    updateTask(id: $id, input: $input) {
      id
      title
      category
      priority
      estimatedMinutes
      notes
      deadlineType
      deadlineDate
      deadlineDays
      isCompleted
    }
  }
`

export const DELETE_TASK = gql`
  mutation DeleteTask($id: UUID!) {
    deleteTask(id: $id)
  }
`

export const COMPLETE_TASK = gql`
  mutation CompleteTask($id: UUID!) {
    completeTask(id: $id) {
      id
      isCompleted
      completedAt
    }
  }
`

export const START_TASK_CONVERSATION = gql`
  mutation StartTaskConversation($message: String!) {
    startTaskConversation(message: $message) {
      id
      status
      messages {
        id
        role
        content
        createdAt
      }
    }
  }
`

export const SEND_TASK_MESSAGE = gql`
  mutation SendTaskMessage($conversationId: UUID!, $message: String!) {
    sendTaskMessage(conversationId: $conversationId, message: $message) {
      id
      status
      messages {
        id
        role
        content
        createdAt
      }
    }
  }
`

export const CONFIRM_TASK_BREAKDOWN = gql`
  mutation ConfirmTaskBreakdown($conversationId: UUID!) {
    confirmTaskBreakdown(conversationId: $conversationId) {
      id
      title
      category
      priority
      parentId
      estimatedMinutes
    }
  }
`
