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
  return (
    <div className="flex min-h-screen">
      {/* Sidebar */}
      <aside className="w-56 bg-white border-r border-gray-200 flex flex-col shadow-sm">
        <div className="px-5 py-4 border-b border-gray-100">
          <h1 className="text-base font-bold text-blue-700">💰 Keuangan Keluarga</h1>
          <p className="text-xs text-gray-500 mt-0.5 truncate">{user?.username}</p>
        </div>
        <nav className="flex-1 py-3 px-2 space-y-0.5">
          {navItems.map(({ to, label, icon }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/'}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors
                 ${isActive
                   ? 'bg-blue-50 text-blue-700 font-medium'
                   : 'text-gray-600 hover:bg-gray-50'}`
              }
            >
              <span>{icon}</span>
              <span>{label}</span>
            </NavLink>
          ))}
        </nav>
        <div className="p-3 border-t border-gray-100">
          <button onClick={onLogout} className="w-full btn-ghost text-sm text-left px-3 py-2 rounded-lg">
            🚪 Keluar
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto bg-gray-50">
        <Outlet />
      </main>
    </div>
  )
}
