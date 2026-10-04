import { useState, useEffect, useCallback } from 'react'
import api, { walletAPI } from '../api.js'
import { rupiah, fmtDate, today, thisMonth, progressColor } from '../utils.js'
import { usePageFocus } from '../hooks/usePageFocus.js'
import { useCurrentUser } from '../App.jsx'

const DOMAINS = [
  { value: 'income',       label: '↑ Pemasukan',      color: 'bg-green-100 text-green-700' },
  { value: 'expense',      label: '↓ Pengeluaran',    color: 'bg-red-100 text-red-600' },
  { value: 'hutang',       label: '🏦 Cicilan Hutang', color: 'bg-purple-100 text-purple-700' },
  { value: 'dana-darurat', label: '🛡 Dana Darurat',   color: 'bg-blue-100 text-blue-700' },
  { value: 'tabungan',     label: '🎯 Tabungan',       color: 'bg-teal-100 text-teal-700' },
]

const EMPTY_TX = {
  domain: 'expense', amount: '', category_name: '', date: today(), note: '',
  debt_id: '', ef_direction: 'setor', saving_id: '', wallet_id: '',
}

// Komponen list yang bisa di-expand (2 item default, selebihnya tersembunyi)
function ExpandableList({ items, renderItem, emptyText, limit = 2 }) {
  const [expanded, setExpanded] = useState(false)
  const visible = expanded ? items : items.slice(0, limit)
  const extra   = items.length - limit

  return (
    <>
      {items.length === 0
        ? <p className="text-xs text-gray-400 italic">{emptyText}</p>
        : visible.map(renderItem)
      }
      {items.length > limit && (
        <button
          onClick={() => setExpanded(e => !e)}
          className="mt-2 text-xs text-blue-500 hover:text-blue-700 font-medium flex items-center gap-1"
        >
          {expanded
            ? <><span>▲</span> Sembunyikan</>
            : <><span>▼</span> {extra} lainnya</>
          }
        </button>
      )}
    </>
  )
}

const WALLET_TYPE_ICON = {
  cash: '💵', bank: '🏦', 'e-wallet': '📱', investment: '📈', other: '🗂',
}

export default function DashboardPage() {
  const [data, setData]   = useState(null)
  const [budgets, setBudgets] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [wallets, setWallets] = useState([])

  const currentUser = useCurrentUser()
  const username    = currentUser?.username || ''

  // Quick-add transaksi
  const [cats, setCats]       = useState([])
  const [debts, setDebts]     = useState([])
  const [savings, setSavings] = useState([])
  const [txForm, setTxForm]   = useState(EMPTY_TX)
  const [txLoading, setTxLoading] = useState(false)
  const [txMsg, setTxMsg]     = useState(null)

  const loadBudgets = useCallback(() => {
    api.get('/budgets', { params: { month: thisMonth() } })
      .then(r => setBudgets(r.data.budgets || []))
      .catch(() => {})
  }, [])

  const load = () => {
    setLoading(true)
    setError(null)
    Promise.all([
      api.get('/dashboard'),
      api.get('/budgets', { params: { month: thisMonth() } }).catch(() => ({ data: { budgets: [] } })),
      walletAPI.list().catch(() => ({ data: { wallets: [] } })),
    ])
      .then(([dashRes, budRes, wallRes]) => {
        setData(dashRes.data)
        setBudgets(budRes.data?.budgets || [])
        setWallets(wallRes.data?.wallets || [])
        setError(null)
      })
      .catch(err => {
        if (err.response?.status !== 401) {
          setError('Gagal memuat dashboard.')
          console.error(err)
        }
      })
      .finally(() => setLoading(false))
  }

  const loadWallets = () => {
    walletAPI.list()
      .then(r => setWallets(r.data.wallets || []))
      .catch(() => {})
  }

  usePageFocus(() => { load(); loadWallets() })

  useEffect(() => {
    api.get('/categories').then(r => setCats(r.data.categories || []))
    api.get('/debts').then(r => setDebts((r.data.debts || []).filter(d => !d.is_paid)))
    api.get('/savings').then(r => setSavings(r.data.savings || []))
    loadWallets()
  }, [])

  const expenseCats = cats.filter(c => c.type === 'expense' || c.type === 'both')
  const incomeCats  = cats.filter(c => c.type === 'income'  || c.type === 'both')
  const domainCats  = txForm.domain === 'income' ? incomeCats : expenseCats

  const handleTxSubmit = async (e) => {
    e.preventDefault()
    setTxLoading(true)
    setTxMsg(null)
    try {
      const { domain, amount, category_name, date, note,
              debt_id, ef_direction, saving_id, wallet_id } = txForm
      const amt = parseFloat(amount)

      if (domain === 'income' || domain === 'expense') {
        await api.post('/transactions', {
          type: domain,
          amount: amt,
          category_name,
          member: username,
          date,
          note,
          wallet_id: wallet_id || undefined,
        })
      } else if (domain === 'hutang') {
        await api.post(`/debts/${debt_id}/pay`, { paid_by: username, amount: amt, date })
        api.get('/debts').then(r => setDebts((r.data.debts || []).filter(d => !d.is_paid)))
      } else if (domain === 'dana-darurat') {
        const signedAmt = ef_direction === 'tarik' ? -Math.abs(amt) : Math.abs(amt)
        await api.post('/budgets/emergency/deposit', { amount: signedAmt, note, date })
      } else if (domain === 'tabungan') {
        await api.post(`/savings/${saving_id}/deposit`, { amount: amt, date, note, member: username })
        api.get('/savings').then(r => setSavings(r.data.savings || []))
      }

      setTxMsg({ type: 'ok', text: 'Berhasil dicatat!' })
      setTxForm(EMPTY_TX)
      load()
      loadWallets()
    } catch (err) {
      setTxMsg({ type: 'err', text: err.response?.data?.error || 'Gagal menyimpan' })
    } finally {
      setTxLoading(false)
    }
  }

  if (loading) return (
    <div className="flex items-center justify-center h-64">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
    </div>
  )

  if (error) return (
    <div className="flex flex-col items-center justify-center h-64 gap-4">
      <p className="text-red-500 text-sm">{error}</p>
      <button onClick={load} className="btn-outline text-sm">Coba Lagi</button>
    </div>
  )

  const tm  = data?.this_month   || {}
  const ef  = data?.emergency_fund
  const efPct = ef && ef.target_balance > 0
    ? Math.min(100, (ef.current_balance / ef.target_balance) * 100)
    : 0
  const totalWalletBalance = wallets.reduce((s, w) => s + (Number(w.balance) || 0), 0)
  const totalBudget = budgets.reduce((s, b) => s + (Number(b.amount) || 0), 0)
  const budgetVsIncomePct = (tm.total_income || 0) > 0 ? (totalBudget / tm.total_income) * 100 : 0
  const totalKasDanPemasukan = totalWalletBalance + (tm.total_income || 0)

  return (
    <div className="p-6 space-y-5">

      {/* ── Header ─────────────────────────────────────────── */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-gray-800">Dashboard</h1>
          <p className="text-xs text-gray-500">Ringkasan keuangan keluarga & integrasi kas dompet</p>
        </div>
        <div className="flex items-center gap-3 bg-indigo-50 border border-indigo-100 px-3.5 py-2 rounded-xl">
          <div className="text-lg">💵</div>
          <div>
            <div className="text-[10px] text-indigo-600 font-semibold uppercase tracking-wider">Total Likuiditas (Dompet + Pemasukan)</div>
            <div className="text-sm font-bold text-indigo-950">{rupiah(totalKasDanPemasukan)}</div>
          </div>
        </div>
      </div>

      {/* ── Ringkasan bulan ini — 6 kartu ────────────────── */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
        <div className="card text-center py-3 border-l-4 border-l-green-500">
          <div className="text-xs text-gray-400 mb-1">Pemasukan</div>
          <div className="text-base font-bold text-green-600 leading-tight">{rupiah(tm.total_income)}</div>
          <div className="text-[10px] text-gray-400 mt-0.5">Bulan ini</div>
        </div>
        <div className="card text-center py-3 border-l-4 border-l-indigo-500">
          <div className="text-xs text-gray-400 mb-1">💰 Total Dompet</div>
          <div className="text-base font-bold text-indigo-600 leading-tight">{rupiah(totalWalletBalance)}</div>
          <div className="text-[10px] text-gray-400 mt-0.5">{wallets.length} dompet aktif</div>
        </div>
        <div className="card text-center py-3">
          <div className="text-xs text-gray-400 mb-1">📋 Total Anggaran</div>
          <div className="text-base font-bold text-amber-600 leading-tight">{rupiah(totalBudget)}</div>
          <div className="text-[10px] text-gray-400 mt-0.5">
            {tm.total_income > 0 ? `${budgetVsIncomePct.toFixed(0)}% dari pemasukan` : `${budgets.length} pos anggaran`}
          </div>
        </div>
        <div className="card text-center py-3">
          <div className="text-xs text-gray-400 mb-1">Pengeluaran</div>
          <div className="text-base font-bold text-red-500 leading-tight">{rupiah(tm.total_expense)}</div>
          <div className="text-[10px] text-gray-400 mt-0.5">
            {totalBudget > 0 ? `${((tm.total_expense / totalBudget) * 100).toFixed(0)}% anggaran` : 'Realisasi'}
          </div>
        </div>
        <div className="card text-center py-3">
          <div className="text-xs text-gray-400 mb-1">Sisa Bulan Ini</div>
          <div className={`text-base font-bold leading-tight ${(tm.balance || 0) >= 0 ? 'text-blue-600' : 'text-red-600'}`}>
            {rupiah(tm.balance)}
          </div>
          <div className="text-[10px] text-gray-400 mt-0.5">Pemasukan - Pengeluaran</div>
        </div>
        <div className="card text-center py-3">
          <div className="text-xs text-gray-400 mb-1">Dana Darurat</div>
          {ef
            ? <>
                <div className={`text-base font-bold leading-tight ${efPct >= 100 ? 'text-green-600' : efPct >= 50 ? 'text-blue-600' : 'text-red-500'}`}>
                  {efPct.toFixed(0)}%
                </div>
                <div className="text-[10px] text-gray-400 mt-0.5">{(ef.months_covered || 0).toFixed(1)} / {ef.target_months ?? 6} bln</div>
              </>
            : <div className="text-base font-bold text-gray-300">—</div>
          }
        </div>
      </div>

      {/* ── Quick-add Transaksi ───────────────────────────── */}
      <div className="card">
        <h2 className="text-sm font-semibold text-gray-700 mb-3">⚡ Catat Transaksi Cepat</h2>
        <form onSubmit={handleTxSubmit} className="space-y-3">
          {/* Domain chips */}
          <div className="flex flex-wrap gap-2">
            {DOMAINS.map(d => (
              <button key={d.value} type="button"
                onClick={() => setTxForm(f => ({ ...f, domain: d.value, category_name: '', debt_id: '', saving_id: '', wallet_id: '' }))}
                className={`px-3 py-1.5 rounded-lg text-xs font-medium border transition-all
                  ${txForm.domain === d.value
                    ? `${d.color} border-current ring-1 ring-current`
                    : 'bg-gray-50 border-gray-200 text-gray-600 hover:bg-gray-100'
                  }`}>
                {d.label}
              </button>
            ))}
          </div>

          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 items-end">
            <div>
              <label className="label">Jumlah (Rp)</label>
              <input className="input" type="number" min="1" step="any" placeholder="0"
                value={txForm.amount}
                onChange={e => setTxForm(f => ({ ...f, amount: e.target.value }))}
                required />
            </div>

            {(txForm.domain === 'income' || txForm.domain === 'expense') && (
              <div>
                <label className="label">Kategori</label>
                <select className="input" value={txForm.category_name}
                  onChange={e => setTxForm(f => ({ ...f, category_name: e.target.value }))} required>
                  <option value="">-- Pilih --</option>
                  {domainCats.map(c => <option key={c.name} value={c.name}>{c.name}</option>)}
                </select>
              </div>
            )}

            {(txForm.domain === 'income' || txForm.domain === 'expense') && wallets.length > 0 && (
              <div>
                <label className="label">Dompet</label>
                <select className="input" value={txForm.wallet_id}
                  onChange={e => setTxForm(f => ({ ...f, wallet_id: e.target.value }))}>
                  <option value="">-- Tanpa Dompet --</option>
                  {wallets.map(w => (
                    <option key={w.id} value={w.id}>{w.name} ({rupiah(w.balance)})</option>
                  ))}
                </select>
              </div>
            )}

            {txForm.domain === 'hutang' && (
              <div>
                <label className="label">Hutang</label>
                <select className="input" value={txForm.debt_id}
                  onChange={e => {
                    const d = debts.find(x => x.id === e.target.value)
                    setTxForm(f => ({ ...f, debt_id: e.target.value, amount: d ? d.monthly_payment : f.amount }))
                  }} required>
                  <option value="">-- Pilih --</option>
                  {debts.map(d => <option key={d.id} value={d.id}>{d.name}</option>)}
                </select>
              </div>
            )}

            {txForm.domain === 'dana-darurat' && (
              <div>
                <label className="label">Jenis</label>
                <select className="input" value={txForm.ef_direction}
                  onChange={e => setTxForm(f => ({ ...f, ef_direction: e.target.value }))}>
                  <option value="setor">⬆ Setor</option>
                  <option value="tarik">⬇ Tarik</option>
                </select>
              </div>
            )}

            {txForm.domain === 'tabungan' && (
              <div>
                <label className="label">Tujuan</label>
                <select className="input" value={txForm.saving_id}
                  onChange={e => setTxForm(f => ({ ...f, saving_id: e.target.value }))} required>
                  <option value="">-- Pilih --</option>
                  {savings.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                </select>
              </div>
            )}

            <div>
              <label className="label">Tanggal</label>
              <input className="input" type="date" value={txForm.date}
                onChange={e => setTxForm(f => ({ ...f, date: e.target.value }))} required />
            </div>

            <div>
              <label className="label opacity-0 select-none">.</label>
              <button type="submit" disabled={txLoading} className="btn-primary w-full">
                {txLoading ? '...' : '+ Simpan'}
              </button>
            </div>
          </div>
        </form>
        {txMsg && (
          <p className={`mt-2 text-xs ${txMsg.type === 'ok' ? 'text-green-600' : 'text-red-500'}`}>
            {txMsg.type === 'ok' ? '✅' : '⚠️'} {txMsg.text}
          </p>
        )}
      </div>

      {/* ── Grid bawah: 4 kolom ──────────────────────────── */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-5">

        {/* Dompet */}
        <div className="card">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-gray-700">💰 Dompet</h3>
            <span className="text-xs text-indigo-600 font-semibold">{rupiah(totalWalletBalance)}</span>
          </div>
          {wallets.length === 0 ? (
            <p className="text-xs text-gray-400 italic">Belum ada dompet</p>
          ) : (
            <ExpandableList
              items={wallets}
              emptyText="Belum ada dompet"
              limit={3}
              renderItem={w => (
                <div key={w.id} className="flex items-center justify-between py-2 border-b border-gray-50 last:border-0">
                  <div className="flex items-center gap-2 min-w-0">
                    <div className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ background: w.color || '#6366f1' }} />
                    <div className="min-w-0">
                      <div className="text-xs font-medium truncate">{w.name}</div>
                      <div className="text-[10px] text-gray-400">{WALLET_TYPE_ICON[w.type] || '🗂'} {w.type}</div>
                    </div>
                  </div>
                  <div className="text-xs font-semibold ml-2 shrink-0" style={{ color: w.color || '#6366f1' }}>
                    {rupiah(Number(w.balance) || 0)}
                  </div>
                </div>
              )}
            />
          )}
        </div>

        {/* Pengeluaran per Kategori */}
        <div className="card">
          <h3 className="text-sm font-semibold text-gray-700 mb-3">Pengeluaran per Kategori</h3>
          <ExpandableList
            items={data?.category_spending || []}
            emptyText="Belum ada pengeluaran bulan ini"
            limit={3}
            renderItem={cs => (
              <div key={cs.category_name} className="mb-2.5">
                <div className="flex justify-between text-xs mb-1">
                  <span className="text-gray-600 truncate">{cs.category_name}</span>
                  <span className="text-gray-500 ml-2 shrink-0">
                    {rupiah(cs.spent)}{cs.budget > 0 ? ` / ${rupiah(cs.budget)}` : ''}
                  </span>
                </div>
                <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
                  <div
                    className={`h-full rounded-full ${progressColor(cs.pct_used || 0)}`}
                    style={{ width: `${Math.min(100, cs.pct_used || 0)}%` }}
                  />
                </div>
              </div>
            )}
          />
          {/* Pengeluaran per anggota — compact chips di bawah */}
          {(data?.member_spending || []).length > 0 && (
            <div className="mt-3 pt-3 border-t border-gray-50 flex flex-wrap gap-1.5">
              {(data.member_spending || []).map(ms => (
                <span key={ms.member} className="badge bg-gray-100 text-gray-600 text-[10px]">
                  {ms.member} {rupiah(ms.spent)}
                </span>
              ))}
            </div>
          )}
        </div>

        {/* Hutang Bulan Ini */}
        <div className="card">
          <h3 className="text-sm font-semibold text-gray-700 mb-2">🏦 Hutang Bulan Ini</h3>
          <div className="flex gap-2 mb-3">
            <div className="flex-1 bg-red-50 rounded-lg px-3 py-2 text-center">
              <div className="text-[10px] text-gray-400">Total Sisa</div>
              <div className="text-xs font-bold text-red-500">{rupiah(data?.total_remaining_debt)}</div>
            </div>
            <div className="flex-1 bg-orange-50 rounded-lg px-3 py-2 text-center">
              <div className="text-[10px] text-gray-400">Cicilan/bln</div>
              <div className="text-xs font-bold text-orange-500">{rupiah(data?.monthly_debt_total)}</div>
            </div>
          </div>
          <ExpandableList
            items={data?.monthly_debts || []}
            emptyText="Tidak ada hutang aktif"
            limit={2}
            renderItem={d => (
              <div key={d.id} className="flex items-center justify-between py-2 border-b border-gray-50 last:border-0">
                <div className="min-w-0">
                  <div className="text-xs font-medium truncate">{d.name}</div>
                  <div className="text-[10px] text-gray-400">Sisa {rupiah(d.remaining_amount)}</div>
                </div>
                <div className="text-right ml-2 shrink-0">
                  <div className="text-[10px] font-semibold text-orange-600">{rupiah(d.monthly_payment)}/bln</div>
                  <div className={`text-[10px] font-medium ${d.days_until_due <= 0 ? 'text-red-600' : d.days_until_due <= 7 ? 'text-orange-600' : 'text-gray-400'}`}>
                    {d.days_until_due <= 0 ? 'Jatuh Tempo!' : `${d.days_until_due}h lagi`}
                  </div>
                </div>
              </div>
            )}
          />
        </div>

        {/* Tujuan Tabungan */}
        <div className="card">
          <h3 className="text-sm font-semibold text-gray-700 mb-3">🎯 Tujuan Tabungan</h3>
          <ExpandableList
            items={data?.savings_summary || []}
            emptyText="Belum ada tujuan tabungan"
            limit={2}
            renderItem={s => (
              <div key={s.id} className="mb-3 last:mb-0">
                <div className="flex justify-between text-xs mb-1">
                  <span className="font-medium truncate">{s.name}</span>
                  <span className="text-gray-500 ml-2 shrink-0">{(s.progress_pct || 0).toFixed(0)}%</span>
                </div>
                <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
                  <div className="h-full bg-green-500 rounded-full"
                    style={{ width: `${Math.min(100, s.progress_pct || 0)}%` }} />
                </div>
                <div className="text-[10px] text-gray-400 mt-0.5">
                  {rupiah(s.current_balance)} / {rupiah(s.target_amount)}
                  {s.monthly_required > 0 && (
                    <span className="ml-1 text-blue-500">· {rupiah(s.monthly_required)}/bln</span>
                  )}
                </div>
              </div>
            )}
          />
        </div>

      </div>
    </div>
  )
}
