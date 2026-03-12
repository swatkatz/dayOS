import { useState, useEffect, useCallback } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ApolloProvider } from '@apollo/client/react'
import { client, getToken, setOnUnauth } from './apollo'
import AuthGate from './components/AuthGate'
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

export default function App() {
  const [authed, setAuthed] = useState(() => !!getToken())

  const handleUnauth = useCallback(() => {
    setAuthed(false)
  }, [])

  useEffect(() => {
    setOnUnauth(handleUnauth)
  }, [handleUnauth])

  useEffect(() => {
    if (authed) {
      requestNotificationPermission()
    }
  }, [authed])

  if (!authed) {
    return <AuthGate onAuth={() => setAuthed(true)} />
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
