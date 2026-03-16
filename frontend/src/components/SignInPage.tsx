import { useState } from 'react'
import { SignIn, SignUp } from '@clerk/clerk-react'

const clerkAppearance = {
  variables: {
    colorBackground: '#1a1a1e',
    colorText: '#e8e6e1',
    colorTextSecondary: '#a0a0a0',
    colorPrimary: '#c5a55a',
    colorInputBackground: '#2a2a2e',
    colorInputText: '#e8e6e1',
    borderRadius: '0.75rem',
  },
  elements: {
    card: { backgroundColor: '#1a1a1e', border: '1px solid #333' },
    headerTitle: { color: '#e8e6e1' },
    headerSubtitle: { color: '#a0a0a0' },
    socialButtonsBlockButton: { backgroundColor: '#2a2a2e', color: '#e8e6e1', border: '1px solid #444' },
    socialButtonsBlockButtonText: { color: '#e8e6e1' },
    formFieldLabel: { color: '#e8e6e1' },
    formFieldInput: { backgroundColor: '#2a2a2e', color: '#e8e6e1', border: '1px solid #444' },
    footerActionLink: { color: '#c5a55a' },
    formButtonPrimary: { backgroundColor: '#c5a55a', color: '#0f0f11' },
    dividerLine: { backgroundColor: '#333' },
    dividerText: { color: '#a0a0a0' },
    identityPreview: { backgroundColor: '#2a2a2e' },
    identityPreviewText: { color: '#e8e6e1' },
    identityPreviewEditButton: { color: '#c5a55a' },
    formFieldAction: { color: '#c5a55a' },
    alertText: { color: '#e8e6e1' },
    footer: { '& span': { color: '#a0a0a0' }, '& a': { color: '#c5a55a' } },
  },
}

export default function SignInPage() {
  const [mode, setMode] = useState<'sign-in' | 'sign-up'>('sign-in')

  return (
    <div className="flex items-center justify-center min-h-screen bg-bg-primary px-4">
      <div className="flex flex-col items-center gap-6">
        <div className="text-center">
          <div className="text-4xl mb-3">🗓️</div>
          <h1 className="text-xl font-semibold text-accent tracking-tight">DayOS</h1>
        </div>
        {mode === 'sign-in' ? (
          <SignIn
            routing="hash"
            signUpUrl="#sign-up"
            appearance={clerkAppearance}
            signUpFallbackRedirectUrl="/"
          />
        ) : (
          <SignUp
            routing="hash"
            signInUrl="#sign-in"
            appearance={clerkAppearance}
            signInFallbackRedirectUrl="/"
          />
        )}
        <button
          onClick={() => setMode(mode === 'sign-in' ? 'sign-up' : 'sign-in')}
          className="text-sm text-text-secondary hover:text-accent transition-colors"
        >
          {mode === 'sign-in' ? "Don't have an account? Sign up" : 'Already have an account? Sign in'}
        </button>
      </div>
    </div>
  )
}
