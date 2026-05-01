import { useEffect } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { configureAuthBridge } from './api/client'
import { useCurrentSession } from './api/hooks'
import { useAuthStore } from './state/authStore'
import { StudioShell } from './studio/layout/StudioShell'
import { AdminSettingsPage } from './studio/pages/AdminSettingsPage'
import { AssetsGraphPage } from './studio/pages/AssetsGraphPage'
import { AuthPage } from './studio/pages/AuthPage'
import { HomePage } from './studio/pages/HomePage'
import { InvitationsPage } from './studio/pages/InvitationsPage'
import { SessionsPage } from './studio/pages/SessionsPage'
import { StoryAnalysisPage } from './studio/pages/StoryAnalysisPage'
import { StoryboardPage } from './studio/pages/StoryboardPage'
import { TimelineExportPage } from './studio/pages/TimelineExportPage'
import { WorkerMetricsPage } from './studio/pages/WorkerMetricsPage'

function App() {
  const session = useAuthStore((state) => state.session)
  const setSession = useAuthStore((state) => state.setSession)
  const clearSession = useAuthStore((state) => state.clearSession)
  const sessionQuery = useCurrentSession(Boolean(session?.token))

  useEffect(() => {
    configureAuthBridge({
      onRefreshed: (next) => setSession(next),
      onCleared: () => clearSession(),
    })
  }, [setSession, clearSession])

  useEffect(() => {
    if (sessionQuery.data) {
      setSession(sessionQuery.data)
    }
  }, [sessionQuery.data, setSession])

  useEffect(() => {
    if (sessionQuery.isError) {
      clearSession()
    }
  }, [clearSession, sessionQuery.isError])

  if (!session?.token) {
    return <AuthPage />
  }

  if (sessionQuery.isLoading) {
    return (
      <main className="auth-loading-shell">
        <div className="auth-loading-card">
          <span className="section-kicker">JWT Session</span>
          <strong>正在恢复导演台会话...</strong>
        </div>
      </main>
    )
  }

  return (
    <Routes>
      <Route element={<StudioShell />}>
        <Route index element={<HomePage />} />
        <Route path="story-analysis" element={<StoryAnalysisPage />} />
        <Route path="storyboard" element={<StoryboardPage />} />
        <Route path="assets-graph" element={<AssetsGraphPage />} />
        <Route path="timeline-export" element={<TimelineExportPage />} />
        <Route path="admin/settings" element={<AdminSettingsPage />} />
        <Route path="admin/invitations" element={<InvitationsPage />} />
        <Route path="admin/worker-metrics" element={<WorkerMetricsPage />} />
        <Route path="account/sessions" element={<SessionsPage />} />
      </Route>
      <Route path="*" element={<Navigate replace to="/" />} />
    </Routes>
  )
}

export default App
