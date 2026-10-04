import { BrowserRouter, Routes, Route, Navigate, useNavigate } from 'react-router-dom'
import { useState, useEffect, useCallback, createContext, useContext } from 'react'
import api, { setNavigateToLogin } from './api.js'

// UserContext — menyediakan username user yang sedang login ke seluruh page
export const UserContext = createContext(null)
export const useCurrentUser = () => useContext(UserContext)

import LoginPage from './pages/LoginPage.jsx'
import RegisterPage from './pages/RegisterPage.jsx'
import DashboardPage from './pages/DashboardPage.jsx'
import TransactionsPage from './pages/TransactionsPage.jsx'
import BudgetsPage from './pages/BudgetsPage.jsx'
import DebtsPage from './pages/DebtsPage.jsx'
import SavingsPage from './pages/SavingsPage.jsx'
import CategoriesPage from './pages/CategoriesPage.jsx'
import EmergencyFundPage from './pages/EmergencyFundPage.jsx'
import AdminPage from './pages/AdminPage.jsx'
import WalletPage from './pages/WalletPage.jsx'
import Layout from './components/Layout.jsx'

// NavigateRegistrar — harus berada di dalam <BrowserRouter> agar useNavigate bekerja.
// onUserClear distabilkan dengan useCallback di App agar effect ini tidak re-run tiap render.
function NavigateRegistrar({ onUserClear }) {
  const navigate = useNavigate()
  useEffect(() => {
    setNavigateToLogin(() => {
      onUserClear()
      navigate('/login', { replace: true })
    })
  }, [navigate, onUserClear])
  return null
}

export default function App() {
  // undefined = sedang cek session, null = tidak login, object = sudah login
  const [user, setUser] = useState(undefined)

  useEffect(() => {
    api.get('/auth/me')
      .then(res => setUser(res.data))
      .catch(() => setUser(null))
  }, [])

  // useCallback agar referensi stabil — NavigateRegistrar tidak re-register tiap render
  const clearUser = useCallback(() => setUser(null), [])

  const handleLogout = async () => {
    await api.post('/auth/logout').catch(() => {})
    setUser(null)
  }

  // Saat session masih dicek, tampilkan spinner di luar <Routes>
  // sehingga Routes tidak di-mount/unmount dan URL tetap terjaga
  if (user === undefined) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    )
  }

  return (
    <UserContext.Provider value={user}>
    <BrowserRouter>
      <NavigateRegistrar onUserClear={clearUser} />

      <Routes>
        <Route path="/login" element={
          user?.user_id
            ? <Navigate to="/" replace />
            : <LoginPage onLogin={setUser} />
        } />
        <Route path="/register" element={
          user?.user_id
            ? <Navigate to="/" replace />
            : <RegisterPage />
        } />

        {/* Protected routes — user sudah pasti non-null di sini */}
        <Route path="/" element={
          user?.user_id
            ? <Layout user={user} onLogout={handleLogout} />
            : <Navigate to="/login" replace />
        }>
          <Route index element={<DashboardPage />} />
          <Route path="transactions" element={<TransactionsPage />} />
          <Route path="wallets" element={<WalletPage />} />
          <Route path="budgets" element={<BudgetsPage />} />
          <Route path="debts" element={<DebtsPage />} />
          <Route path="savings" element={<SavingsPage />} />
          <Route path="categories" element={<CategoriesPage />} />
          <Route path="emergency-fund" element={<EmergencyFundPage />} />
          <Route path="admin" element={<AdminPage />} />
        </Route>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
    </UserContext.Provider>
  )
}
