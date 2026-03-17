import { useState } from 'react'
import { SignIn, SignUp } from '@clerk/clerk-react'

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
            signUpFallbackRedirectUrl="/"
          />
        ) : (
          <SignUp
            routing="hash"
            signInUrl="#sign-in"
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
