import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import Layout from './Layout'
import Dashboard from './pages/Dashboard'
import CreateInstance from './pages/CreateInstance'

// Placeholder pages
const Instances = () => <Navigate to="/" replace />
const Billing = () => <div className="text-muted-foreground">Billing Module Loading...</div>
const Settings = () => <div className="text-muted-foreground">Settings Module Loading...</div>

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/instances" element={<Instances />} />
          <Route path="/instances/new" element={<CreateInstance />} />
          <Route path="/billing" element={<Billing />} />
          <Route path="/settings" element={<Settings />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App
