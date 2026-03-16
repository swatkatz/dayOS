// Deterministic daily message — same message all day, changes each day
const MESSAGES = [
  'Small steps lead to big wins.',
  'You don\'t have to be perfect, just present.',
  'Progress, not perfection.',
  'One block at a time.',
  'You\'re building momentum.',
  'Today is a fresh start.',
  'Trust the process.',
  'Done is better than perfect.',
  'Focus on what matters most.',
  'You\'ve got this.',
  'Consistency beats intensity.',
  'Show up and start.',
  'Every task completed is a small victory.',
  'Your future self will thank you.',
  'The hardest part is starting.',
  'Keep going — you\'re doing great.',
  'Breathe. Focus. Execute.',
  'Make today count.',
  'Momentum is built one block at a time.',
  'You\'re closer than you think.',
  'Celebrate the small wins.',
  'Energy flows where focus goes.',
  'Do less, but do it well.',
  'You set the pace today.',
  'Intentional days make an intentional life.',
  'Start with what\'s in front of you.',
  'Little by little, a little becomes a lot.',
  'Your plan is your compass, not your cage.',
  'Rest is productive too.',
  'Be kind to yourself today.',
  'What gets planned gets done.',
]

export function useDailyMotivation(): string {
  const now = new Date()
  // Day-based seed: same message all day
  const dayIndex = Math.floor(now.getTime() / (1000 * 60 * 60 * 24))
  return MESSAGES[dayIndex % MESSAGES.length]
}
