import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { ClerkProvider } from '@clerk/clerk-react'
import './index.css'
import App from './App.tsx'

const publishableKey = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY

if (!publishableKey) {
  throw new Error('Missing VITE_CLERK_PUBLISHABLE_KEY environment variable')
}

// Global Clerk dark theme — applies to all Clerk-rendered UI (sign-in, sign-up,
// OAuth errors, account portal, verification screens, etc.)
const clerkAppearance = {
  variables: {
    colorBackground: '#1a1a1e',
    colorText: '#e8e6e1',
    colorTextSecondary: '#a0a0a0',
    colorPrimary: '#c5a55a',
    colorInputBackground: '#2a2a2e',
    colorInputText: '#e8e6e1',
    colorNeutral: '#a0a0a0',
    borderRadius: '0.75rem',
  },
  elements: {
    card: { backgroundColor: '#1a1a1e', border: '1px solid #333' },
    headerTitle: { color: '#e8e6e1' },
    headerSubtitle: { color: '#a0a0a0' },
    socialButtonsBlockButton: { backgroundColor: '#2a2a2e', color: '#e8e6e1', border: '1px solid #444' },
    socialButtonsBlockButtonText: { color: '#e8e6e1' },
    socialButtonsProviderIcon: { filter: 'invert(1)' },
    formFieldLabel: { color: '#e8e6e1' },
    formFieldInput: { backgroundColor: '#2a2a2e', color: '#e8e6e1', border: '1px solid #444' },
    formFieldInputShowPasswordButton: { color: '#a0a0a0' },
    formFieldHintText: { color: '#a0a0a0' },
    footerActionLink: { color: '#c5a55a' },
    formButtonPrimary: { backgroundColor: '#c5a55a', color: '#0f0f11' },
    dividerLine: { backgroundColor: '#333' },
    dividerText: { color: '#a0a0a0' },
    identityPreview: { backgroundColor: '#2a2a2e' },
    identityPreviewText: { color: '#e8e6e1' },
    identityPreviewEditButton: { color: '#c5a55a' },
    formFieldAction: { color: '#c5a55a' },
    alert: { backgroundColor: '#2a2a2e', border: '1px solid #444' },
    alertText: { color: '#e8e6e1' },
    badge: { backgroundColor: '#2a2a2e', color: '#e8e6e1', border: '1px solid #444' },
    otpCodeFieldInput: { backgroundColor: '#2a2a2e', color: '#e8e6e1', border: '1px solid #444' },
    otpCodeField: { color: '#e8e6e1' },
    formResendCodeLink: { color: '#c5a55a' },
    footer: { '& span': { color: '#a0a0a0' }, '& a': { color: '#c5a55a' } },
  },
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ClerkProvider publishableKey={publishableKey} appearance={clerkAppearance}>
      <App />
    </ClerkProvider>
  </StrictMode>,
)
