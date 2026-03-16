interface ReplanBannerProps {
  onReplan: () => void
  onDismiss: () => void
}

export default function ReplanBanner({ onReplan, onDismiss }: ReplanBannerProps) {
  return (
    <div className="bg-accent-dim border border-accent/20 rounded-xl px-4 py-3 mb-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <span className="text-text-primary text-sm">
        Your calendar has changed. Replan remaining blocks?
      </span>
      <div className="flex items-center gap-2 flex-shrink-0">
        <button
          onClick={onDismiss}
          className="px-3 py-1.5 text-text-secondary hover:text-text-primary text-sm transition-colors"
        >
          Dismiss
        </button>
        <button
          onClick={onReplan}
          className="px-3 py-1.5 bg-accent text-black rounded-lg text-sm font-medium hover:bg-accent-hover active:scale-95 transition-all"
        >
          Replan
        </button>
      </div>
    </div>
  )
}
