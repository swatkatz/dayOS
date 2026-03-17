# DayOS

**Your AI-powered daily planner.** Tell Claude about your day, get a structured schedule, then execute it block by block.

DayOS combines AI planning with hands-on time management. Instead of manually dragging blocks around a calendar, you describe your day in natural language and the AI builds a schedule around your routines, tasks, deadlines, and calendar events.

---

## Quick Guide

### 1. Set Up Your Context

**Context** is what the AI knows about you. Go to the **Context** page and add entries about your life, constraints, and preferences.

| Category | Example |
|---|---|
| **Life** | "I have a 6-month-old baby" |
| **Constraints** | "I work 9-5, commute takes 30 min" |
| **Equipment** | "Home gym with dumbbells and pull-up bar" |
| **Preferences** | "I do my best deep work before noon" |

You can also **connect your Google Calendar** here. DayOS will pull in your events automatically and plan around them.

### 2. Define Your Routines

**Routines** are recurring activities the AI should schedule every day (or on specific days). Go to the **Routines** page and add things like:

- Morning workout (daily, 45 min, morning)
- Baby bath time (daily, 30 min, 6:30 PM)
- Team standup (weekdays, 15 min, 9:00 AM)

Each routine has a frequency (daily, weekdays, weekly, or custom days), a preferred time slot or exact time, and an estimated duration.

### 3. Build Your Backlog

The **Backlog** is your task list. Add one-off tasks with categories, priorities, and deadlines. Tasks can be standalone or grouped under parent tasks with subtasks.

**Scope with AI:** Have a big goal like "Redesign the landing page"? Click **Scope with AI** and describe it. The AI will ask clarifying questions, then propose a parent task with estimated subtasks. Confirm to create them all at once.

### 4. Plan Your Day

This is where the magic happens. Go to the **Today** page and start chatting:

> "I have a dentist appointment at 2pm that'll take an hour. I want to make progress on the API migration and cook dinner tonight."

The AI generates a time-blocked schedule on the right panel. It factors in:
- Your routines for today
- Calendar events
- Backlog tasks (prioritizing deadlines and frequently-deferred items)
- Your context and preferences

**Iterate naturally:** "Move exercise to the evening." "Add a 15-minute break after lunch." "I don't have time for the report today." Each message regenerates the plan.

**Edit manually** before accepting: drag to reorder, click to edit times/durations/titles, add or remove blocks.

**Accept** when you're happy. The plan locks in and you enter execution mode.

### 5. Execute Your Plan

Once accepted, your day is a checklist of time blocks:

- **Complete** a block when you finish it (the last one triggers confetti)
- **Skip** a block if plans change (skipped tasks carry over to tomorrow's review)
- **Adjust duration** by clicking the time badge
- **Replan** if something major changes — hit "Something came up" to go back to chat

If your Google Calendar is connected, DayOS polls for changes every 15 minutes and shows a **Replan** banner if your calendar has shifted.

### 6. Plan Tomorrow

Use the **Today/Tomorrow** toggle to plan ahead. Tomorrow's plan works the same way — chat with the AI, preview blocks, accept — but blocks are read-only until that day arrives.

### 7. Review Skipped Tasks

Each morning, if yesterday's plan had skipped tasks, DayOS shows a **carry-over review** before you can start planning. For each skipped task, choose to carry it forward (it stays in the backlog) or mark it done.

### 8. Check Your History

The **History** page shows all past plans with their blocks and chat logs. See what you planned vs. what you actually did, how many blocks were skipped, and how your planning accuracy improves over time.

---

## Tech Stack

Go + GraphQL + PostgreSQL backend, React + TypeScript frontend, deployed on Railway. Auth via Clerk (invite-only). AI powered by Claude.
