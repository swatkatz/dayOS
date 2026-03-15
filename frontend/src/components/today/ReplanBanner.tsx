interface ReplanBannerProps {
  onReplan: () => void
  onDismiss: () => void
}

export default function ReplanBanner({ onReplan, onDismiss }: ReplanBannerProps) {
  return (
    <div className="bg-bg-surface border border-accent/30 rounded-lg px-4 py-3 mb-4 flex items-center justify-between gap-4">
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
          className="px-3 py-1.5 bg-accent text-black rounded text-sm font-medium hover:bg-accent-hover transition-colors"
        >
          Replan
        </button>
      </div>
    </div>
  )
}
