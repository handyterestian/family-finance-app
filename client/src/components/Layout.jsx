import { useState } from 'react'
import { Outlet, NavLink } from 'react-router-dom'

const navItems = [
  { to: '/',                label: 'Dashboard',    icon: '📊' },
  { to: '/transactions',    label: 'Transaksi',    icon: '💳' },
  { to: '/wallets',         label: 'Dompet',       icon: '💰' },
  { to: '/budgets',         label: 'Anggaran',     icon: '📋' },
  { to: '/emergency-fund',  label: 'Dana Darurat', icon: '🛡' },
  { to: '/debts',           label: 'Hutang',       icon: '🏦' },
  { to: '/savings',         label: 'Tabungan',     icon: '🎯' },
  { to: '/categories',      label: 'Kategori',     icon: '🏷️'  },
  { to: '/admin',           label: 'Admin',        icon: '⚙️'  },
]

export default function Layout({ user, onLogout }) {
  const [sidebarOpen, setSidebarOpen] = useState(false)

  const navLinkClass = ({ isActive }) =>
    `flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors
     ${isActive
       ? 'bg-blue-50 text-blue-700 font-medium'
       : 'text-gray-600 hover:bg-gray-50'}`

  return (
    <div className="flex min-h-screen">

      {/* ── Mobile overlay backdrop ──────────────────────────── */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-30 bg-black/40 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* ── Sidebar ──────────────────────────────────────────── */}
      <aside className={`
        fixed inset-y-0 left-0 z-40 w-56 bg-white border-r border-gray-200 flex flex-col shadow-sm
        transform transition-transform duration-200
        lg:static lg:translate-x-0
        ${sidebarOpen ? 'translate-x-0' : '-translate-x-full'}
      `}>
        {/* Brand */}
        <div className="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
          <div>
            <h1 className="text-base font-bold text-blue-700">💰 Keuangan Keluarga</h1>
            <p className="text-xs text-gray-500 mt-0.5 truncate">{user?.username}</p>
          </div>
          {/* Close button — mobile only */}
          <button
            onClick={() => setSidebarOpen(false)}
            className="lg:hidden p-1 rounded text-gray-400 hover:text-gray-600"
          >
            ✕
          </button>
        </div>

        {/* Nav links */}
        <nav className="flex-1 py-3 px-2 space-y-0.5 overflow-y-auto">
          {navItems.map(({ to, label, icon }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/'}
              className={navLinkClass}
              onClick={() => setSidebarOpen(false)}
            >
              <span>{icon}</span>
              <span>{label}</span>
            </NavLink>
          ))}
        </nav>

        {/* Logout */}
        <div className="p-3 border-t border-gray-100">
          <button onClick={onLogout} className="w-full btn-ghost text-sm text-left px-3 py-2 rounded-lg">
            🚪 Keluar
          </button>
        </div>
      </aside>

      {/* ── Main content ─────────────────────────────────────── */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Mobile top bar */}
        <header className="lg:hidden sticky top-0 z-20 bg-white border-b border-gray-200 flex items-center gap-3 px-4 py-3 shadow-sm">
          <button
            onClick={() => setSidebarOpen(true)}
            className="p-1.5 rounded-lg text-gray-500 hover:bg-gray-100"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </button>
          <span className="text-sm font-bold text-blue-700">💰 Keuangan Keluarga</span>
        </header>

        <main className="flex-1 overflow-auto bg-gray-50">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
