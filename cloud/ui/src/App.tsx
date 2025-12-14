import { BrowserRouter, Routes, Route } from 'react-router-dom'
import Layout from './Layout'
import Dashboard from './pages/Dashboard'
import CreateInstance from './pages/CreateInstance'
import Login from './pages/Login'
import Billing from './pages/Billing'
import Settings from './pages/Settings'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Public routes */}
        <Route path="/login" element={<Login />} />

        {/* Protected routes with layout */}
        <Route element={<Layout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/instances" element={<Dashboard />} />
          <Route path="/instances/new" element={<CreateInstance />} />
          <Route path="/billing" element={<Billing />} />
          <Route path="/settings" element={<Settings />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App
