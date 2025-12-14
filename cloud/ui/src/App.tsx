import { useState, useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Toaster } from 'sonner'
import { AuthProvider } from './hooks/useAuth'
import Layout from './Layout'
import Dashboard from './pages/Dashboard'
import CreateInstance from './pages/CreateInstance'
import InstanceDetail from './pages/InstanceDetail'
import Login from './pages/Login'
import Register from './pages/Register'
import Billing from './pages/Billing'
import Settings from './pages/Settings'
import Onboarding from './components/Onboarding'
import { ThemeProvider } from './components/ThemeToggle'

function AppContent() {
  const [showOnboarding, setShowOnboarding] = useState(false)

  useEffect(() => {
    // Check if user has completed onboarding
    const completed = localStorage.getItem('onboarding_completed')
    // const token = localStorage.getItem('access_token') // Unused

    // Show onboarding for new users (not completed and not on auth pages)
    if (!completed && !window.location.pathname.includes('/login') && !window.location.pathname.includes('/register')) {
      setShowOnboarding(true)
    }
  }, [])

  const handleOnboardingComplete = () => {
    setShowOnboarding(false)
  }

  return (
    <>
      {/* Toast notifications */}
      <Toaster
        position="top-right"
        toastOptions={{
          className: 'bg-card border border-border text-foreground',
        }}
        richColors
      />

      {/* Onboarding wizard for new users */}
      {showOnboarding && (
        <Onboarding
          onComplete={handleOnboardingComplete}
          onMinimize={() => setShowOnboarding(false)}
        />
      )}

      <Routes>
        {/* Public routes */}
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />

        {/* Protected routes with layout */}
        <Route element={<Layout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/instances" element={<Dashboard />} />
          <Route path="/instances/new" element={<CreateInstance />} />
          <Route path="/instances/:id" element={<InstanceDetail />} />
          <Route path="/billing" element={<Billing />} />
          <Route path="/settings" element={<Settings />} />
        </Route>

        {/* Catch all */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </>
  )
}

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter>
          <AppContent />
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  )
}

export default App
