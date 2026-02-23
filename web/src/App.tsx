import './App.css'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext'
import Login from './components/Login'
import OAuthCallback from './components/OauthCallback'
import ProtectedRoute from './components/ProtectedRoute'
import Dashboard from './components/Dashboard'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from './lib/queryClient'

function App() {
  return (
    <BrowserRouter>
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/oauth-callback" element={<OAuthCallback />} />
          <Route
            path="/dashboard"
            element={
              <ProtectedRoute>
                <Dashboard />
              </ProtectedRoute>
            }
          />
          <Route path="/" element={<Navigate to="/dashboard" />} />
        </Routes>
      </AuthProvider>
      </QueryClientProvider>
    </BrowserRouter>
  )
}

export default App