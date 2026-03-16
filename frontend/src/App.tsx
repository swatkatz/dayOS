import { useState, useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ApolloProvider } from '@apollo/client/react'
import { SignedIn, SignedOut, useAuth, useClerk } from '@clerk/clerk-react'
import { client, setTokenGetter, setOnUnauth } from './apollo'
import SignInPage from './components/SignInPage'
import Layout from './components/Layout'
import TodayPage from './pages/TodayPage'
import BacklogPage from './pages/BacklogPage'
import RoutinesPage from './pages/RoutinesPage'
import ContextPage from './pages/ContextPage'
import HistoryPage from './pages/HistoryPage'

function requestNotificationPermission() {
  if (typeof Notification !== 'undefined' && Notification.permission === 'default') {
    Notification.requestPermission()
  }
}

function AuthenticatedApp() {
  const { getToken } = useAuth()
  const { signOut } = useClerk()
  const [ready, setReady] = useState(false)

  useEffect(() => {
    setTokenGetter(() => getToken())
    setOnUnauth(() => signOut())
    setReady(true)
  }, [getToken, signOut])

  useEffect(() => {
    if (ready) {
      requestNotificationPermission()
    }
  }, [ready])

  if (!ready) {
    return null
  }

  return (
    <ApolloProvider client={client}>
      <BrowserRouter>
        <Routes>
          <Route element={<Layout />}>
            <Route path="/" element={<TodayPage />} />
            <Route path="/backlog" element={<BacklogPage />} />
            <Route path="/routines" element={<RoutinesPage />} />
            <Route path="/context" element={<ContextPage />} />
            <Route path="/history" element={<HistoryPage />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ApolloProvider>
  )
}

export default function App() {
  return (
    <>
      <SignedOut>
        <SignInPage />
      </SignedOut>
      <SignedIn>
        <AuthenticatedApp />
      </SignedIn>
    </>
  )
}
