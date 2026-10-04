import { useState, useEffect, useCallback } from 'react'
import api from '../api.js'
import { rupiah, fmtDate, today } from '../utils.js'
import Modal from '../components/Modal.jsx'
import { usePageFocus } from '../hooks/usePageFocus.js'

// Progress ring SVG sederhana
function ProgressRing({ pct, size = 120, stroke = 10 }) {
  const r = (size - stroke) / 2
  const circ = 2 * Math.PI * r
  const offset = circ - Math.min(1, pct / 100) * circ
  const color = pct >= 100 ? '#22c55e' : pct >= 66 ? '#3b82f6' : pct >= 33 ? '#f59e0b' : '#ef4444'
  return (
    <svg width={size} height={size} className="rotate-[-90deg]">
      <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="#f3f4f6" strokeWidth={stroke} />
      <circle
        cx={size / 2} cy={size / 2} r={r} fill="none"
        stroke={color} strokeWidth={stroke}
        strokeDasharray={circ} strokeDashoffset={offset}
        strokeLinecap="round"
        style={{ transition: 'stroke-dashoffset 0.5s ease' }}
      />
    </svg>
  )
}

export default function EmergencyFundPage() {
  const [ef, setEf] = useState(null)
  const [history, setHistory] = useState([])
  const [loadingEf, setLoadingEf] = useState(true)

  // Modal: deposit / tarik
  const [showDepositModal, setShowDepositModal] = useState(false)
  const [depositType, setDepositType] = useState('setor') // 'setor' | 'tarik'
  const [depositForm, setDepositForm] = useState({ amount: '', note: '', date: today() })
  const [depositLoading, setDepositLoading] = useState(false)

  // Modal: pengaturan (target saldo & target bulan)
  const [showSettingsModal, setShowSettingsModal] = useState(false)
  const [settingsForm, setSettingsForm] = useState({ current_balance: '', target_months: 6 })
  const [settingsLoading, setSettingsLoading] = useState(false)

  const loadAll = useCallback(async () => {
    setLoadingEf(true)
    try {
      const [efRes, histRes] = await Promise.all([
        api.get('/budgets/emergency'),
        api.get('/budgets/emergency/history'),
      ])
      setEf(efRes.data.emergency_fund)
      setHistory(histRes.data.history || [])
    } finally {
      setLoadingEf(false)
    }
  }, [])

  useEffect(() => { loadAll() }, [loadAll])
  usePageFocus(loadAll)

  // ── Deposit / Tarik ──────────────────────────────────────
  const openDeposit = (type) => {
    setDepositType(type)
    setDepositForm({ amount: '', note: '', date: today() })
    setShowDepositModal(true)
  }

  const handleDepositSubmit = async (e) => {
    e.preventDefault()
    setDepositLoading(true)
    try {
      const rawAmount = parseFloat(depositForm.amount)
      const amount = depositType === 'tarik' ? -Math.abs(rawAmount) : Math.abs(rawAmount)
      await api.post('/budgets/emergency/deposit', {
        amount,
        note: depositForm.note,
        date: depositForm.date,
      })
      setShowDepositModal(false)
      loadAll()
    } catch (err) {
      alert(err.response?.data?.error || 'Gagal mencatat transaksi')
    } finally {
      setDepositLoading(false)
    }
  }

  // ── Pengaturan ────────────────────────────────────────────
  const openSettings = () => {
    setSettingsForm({
      current_balance: ef?.current_balance ?? '',
      target_months: ef?.target_months ?? 6,
    })
    setShowSettingsModal(true)
  }

  const handleSettingsSubmit = async (e) => {
    e.preventDefault()
    setSettingsLoading(true)
    try {
      await api.put('/budgets/emergency', {
        current_balance: parseFloat(settingsForm.current_balance),
        target_months: parseInt(settingsForm.target_months, 10),
      })
      setShowSettingsModal(false)
      loadAll()
    } catch (err) {
      alert(err.response?.data?.error || 'Gagal menyimpan pengaturan')
    } finally {
      setSettingsLoading(false)
    }
  }

  // ── Kalkulasi tampilan ─────────────────────────────────────
  const progressPct = ef && ef.target_balance > 0
    ? Math.min(100, (ef.current_balance / ef.target_balance) * 100)
    : 0
  const progressColor = progressPct >= 100 ? 'text-green-600' : progressPct >= 66 ? 'text-blue-600' : progressPct >= 33 ? 'text-yellow-500' : 'text-red-500'
  const statusLabel = progressPct >= 100 ? '✅ Target Tercapai' : progressPct >= 66 ? '🔵 Dalam Jalur' : progressPct >= 33 ? '⚠️ Perlu Ditingkatkan' : '🔴 Kritis'
  const targetMonths = ef?.target_months ?? 6

  if (loadingEf) return (
    <div className="flex items-center justify-center h-64">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
    </div>
  )

  return (
    <div className="p-4 sm:p-6 space-y-5 max-w-3xl mx-auto">

      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-gray-800">🛡 Dana Darurat</h1>
        <button onClick={openSettings} className="btn-outline text-sm">⚙️ Pengaturan</button>
      </div>

      {/* Card utama — progress ring + info */}
      <div className="card">
        <div className="flex flex-col sm:flex-row sm:items-center gap-6 sm:gap-8">
          {/* Ring */}
          <div className="relative flex-shrink-0 self-center">
            <ProgressRing pct={progressPct} size={140} stroke={12} />
            <div className="absolute inset-0 flex flex-col items-center justify-center">
              <span className={`text-xl font-bold ${progressColor}`}>{progressPct.toFixed(0)}%</span>
              <span className="text-xs text-gray-400">tercapai</span>
            </div>
          </div>

          {/* Detail */}
          <div className="flex-1 space-y-3">
            <div>
              <div className="text-xs text-gray-500 mb-0.5">Saldo Saat Ini</div>
              <div className="text-2xl font-bold text-gray-800">{rupiah(ef?.current_balance ?? 0)}</div>
            </div>
            <div className="grid grid-cols-2 gap-3 text-sm">
              <div className="bg-gray-50 rounded-lg p-2.5">
                <div className="text-xs text-gray-500">Target ({targetMonths} bulan)</div>
                <div className="font-semibold text-gray-700">{rupiah(ef?.target_balance ?? 0)}</div>
              </div>
              <div className="bg-gray-50 rounded-lg p-2.5">
                <div className="text-xs text-gray-500">Tercover</div>
                <div className={`font-semibold ${progressColor}`}>
                  {(ef?.months_covered ?? 0).toFixed(1)} / {targetMonths} bulan
                </div>
              </div>
              <div className="bg-gray-50 rounded-lg p-2.5">
                <div className="text-xs text-gray-500">Rata-rata Pengeluaran</div>
                <div className="font-semibold text-gray-700">{rupiah(ef?.monthly_expense_avg ?? 0)} / bln</div>
              </div>
              <div className="bg-gray-50 rounded-lg p-2.5">
                <div className="text-xs text-gray-500">Status</div>
                <div className="font-semibold text-sm">{statusLabel}</div>
              </div>
            </div>

            {/* Sisa yang perlu dikumpulkan */}
            {progressPct < 100 && (ef?.target_balance ?? 0) > (ef?.current_balance ?? 0) && (
              <div className="text-xs text-gray-500">
                Sisa yang perlu dikumpulkan:{' '}
                <span className="font-semibold text-blue-600">
                  {rupiah((ef?.target_balance ?? 0) - (ef?.current_balance ?? 0))}
                </span>
              </div>
            )}
          </div>
        </div>

        {/* Tombol aksi */}
        <div className="flex gap-3 mt-5 pt-4 border-t border-gray-100">
          <button onClick={() => openDeposit('setor')} className="btn-primary flex-1 text-sm">
            ⬆ Setor Dana
          </button>
          <button
            onClick={() => openDeposit('tarik')}
            disabled={!ef || ef.current_balance <= 0}
            className="btn-outline flex-1 text-sm text-red-600 border-red-200 hover:bg-red-50"
          >
            ⬇ Tarik Dana
          </button>
        </div>
      </div>

      {/* Panduan dana darurat */}
      <div className="card bg-blue-50 border border-blue-100">
        <h3 className="text-sm font-semibold text-blue-800 mb-2">💡 Panduan Dana Darurat</h3>
        <ul className="text-xs text-blue-700 space-y-1">
          <li>• Idealnya dana darurat mencukupi <strong>{targetMonths} bulan</strong> pengeluaran keluarga</li>
          <li>• Simpan di rekening terpisah yang mudah dicairkan, bukan investasi jangka panjang</li>
          <li>• Gunakan hanya untuk keadaan darurat: PHK, sakit mendadak, perbaikan mendesak</li>
          <li>• Segera isi kembali setelah digunakan</li>
        </ul>
      </div>

      {/* Riwayat Transaksi */}
      <div className="card">
        <h3 className="text-sm font-semibold text-gray-700 mb-3">📋 Riwayat Transaksi</h3>
        {history.length === 0 ? (
          <p className="text-xs text-gray-400 py-3 text-center">Belum ada riwayat setoran atau penarikan</p>
        ) : (
          <div className="space-y-0 divide-y divide-gray-50">
            {history.map(h => {
              const isDeposit = h.amount > 0
              return (
                <div key={h.id} className="flex items-center justify-between py-2.5">
                  <div className="flex items-center gap-3">
                    <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm flex-shrink-0 ${isDeposit ? 'bg-green-50 text-green-600' : 'bg-red-50 text-red-500'}`}>
                      {isDeposit ? '⬆' : '⬇'}
                    </div>
                    <div>
                      <div className="text-sm font-medium text-gray-800">
                        {isDeposit ? 'Setoran' : 'Penarikan'}
                        {h.note && <span className="text-gray-500 font-normal"> — {h.note}</span>}
                      </div>
                      <div className="text-xs text-gray-400">{fmtDate(h.date)}</div>
                    </div>
                  </div>
                  <div className={`text-sm font-semibold ${isDeposit ? 'text-green-600' : 'text-red-500'}`}>
                    {isDeposit ? '+' : ''}{rupiah(h.amount)}
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* Modal Setor / Tarik */}
      {showDepositModal && (
        <Modal
          title={depositType === 'setor' ? '⬆ Setor Dana Darurat' : '⬇ Tarik Dana Darurat'}
          onClose={() => setShowDepositModal(false)}
        >
          <form onSubmit={handleDepositSubmit} className="space-y-3">
            {depositType === 'tarik' && (
              <div className="bg-red-50 text-red-700 text-xs rounded-lg p-2.5">
                Saldo saat ini: <strong>{rupiah(ef?.current_balance ?? 0)}</strong>
              </div>
            )}
            <div>
              <label className="label">Jumlah (Rp)</label>
              <input
                className="input" type="number" min="1" step="any"
                value={depositForm.amount}
                onChange={e => setDepositForm(f => ({ ...f, amount: e.target.value }))}
                required
              />
            </div>
            <div>
              <label className="label">Tanggal</label>
              <input
                className="input" type="date"
                value={depositForm.date}
                onChange={e => setDepositForm(f => ({ ...f, date: e.target.value }))}
                required
              />
            </div>
            <div>
              <label className="label">Catatan (opsional)</label>
              <input
                className="input" type="text" placeholder="mis. gaji bulan ini"
                value={depositForm.note}
                onChange={e => setDepositForm(f => ({ ...f, note: e.target.value }))}
              />
            </div>
            <div className="flex gap-2 pt-1">
              <button type="submit" disabled={depositLoading}
                className={`flex-1 ${depositType === 'tarik' ? 'bg-red-500 hover:bg-red-600 text-white rounded-lg px-4 py-2 text-sm font-medium' : 'btn-primary'}`}>
                {depositLoading ? 'Menyimpan...' : (depositType === 'setor' ? 'Setor' : 'Tarik')}
              </button>
              <button type="button" onClick={() => setShowDepositModal(false)} className="btn-outline flex-1">Batal</button>
            </div>
          </form>
        </Modal>
      )}

      {/* Modal Pengaturan */}
      {showSettingsModal && (
        <Modal title="⚙️ Pengaturan Dana Darurat" onClose={() => setShowSettingsModal(false)}>
          <form onSubmit={handleSettingsSubmit} className="space-y-3">
            <div>
              <label className="label">Saldo Awal / Koreksi Saldo (Rp)</label>
              <input
                className="input" type="number" min="0" step="any"
                value={settingsForm.current_balance}
                onChange={e => setSettingsForm(f => ({ ...f, current_balance: e.target.value }))}
                required
              />
              <p className="text-xs text-gray-400 mt-1">
                Gunakan ini untuk koreksi saldo langsung. Untuk mencatat setoran/penarikan, gunakan tombol Setor/Tarik.
              </p>
            </div>
            <div>
              <label className="label">Target Bulan Pengeluaran</label>
              <select
                className="input"
                value={settingsForm.target_months}
                onChange={e => setSettingsForm(f => ({ ...f, target_months: e.target.value }))}
              >
                {[3, 6, 9, 12].map(m => (
                  <option key={m} value={m}>{m} bulan {m === 6 ? '(direkomendasikan)' : ''}</option>
                ))}
              </select>
              <p className="text-xs text-gray-400 mt-1">
                Rata-rata pengeluaran akan dihitung otomatis dari 3 bulan terakhir.
              </p>
            </div>
            <div className="flex gap-2 pt-1">
              <button type="submit" disabled={settingsLoading} className="btn-primary flex-1">
                {settingsLoading ? 'Menyimpan...' : 'Simpan'}
              </button>
              <button type="button" onClick={() => setShowSettingsModal(false)} className="btn-outline flex-1">Batal</button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  )
}
