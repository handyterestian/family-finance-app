import { useState, useEffect, useCallback } from 'react'
import api, { walletAPI } from '../api.js'
import { rupiah, fmtDate, today, progressColor } from '../utils.js'
import Modal from '../components/Modal.jsx'
import { usePageFocus } from '../hooks/usePageFocus.js'
import { useCurrentUser } from '../App.jsx'

// Domain jenis transaksi
const DOMAINS = [
  { value: 'income',        label: '↑ Pemasukan',    color: 'bg-green-100 text-green-700' },
  { value: 'expense',       label: '↓ Pengeluaran',  color: 'bg-red-100 text-red-600' },
  { value: 'hutang',        label: '🏦 Cicilan Hutang', color: 'bg-purple-100 text-purple-700' },
  { value: 'dana-darurat',  label: '🛡 Dana Darurat', color: 'bg-blue-100 text-blue-700' },
  { value: 'tabungan',      label: '🎯 Tabungan',     color: 'bg-teal-100 text-teal-700' },
]

const EMPTY_FORM = {
  domain: 'expense',
  amount: '',
  category_name: '',
  date: today(),
  note: '',
  // hutang
  debt_id: '',
  // dana-darurat
  ef_direction: 'setor', // 'setor' | 'tarik'
  // tabungan
  saving_id: '',
  // dompet
  wallet_id: '',
}

export default function TransactionsPage() {
  const currentUser = useCurrentUser()
  const username = currentUser?.username || ''

  const [txs, setTxs]        = useState([])
  const [cats, setCats]       = useState([])
  const [debts, setDebts]     = useState([])
  const [savings, setSavings]  = useState([])
  const [wallets, setWallets]  = useState([])
  const [budgets, setBudgets]  = useState([])
  const [month, setMonth]     = useState(new Date().toISOString().substring(0, 7))
  const [showForm, setShowForm] = useState(false)
  const [editTx, setEditTx]   = useState(null)
  const [form, setForm]       = useState(EMPTY_FORM)
  const [loading, setLoading]  = useState(false)
  const [budgetOpen, setBudgetOpen] = useState(true)  // panel expand/collapse

  const loadWallets = useCallback(() => {
    walletAPI.list().then(r => setWallets(r.data.wallets || [])).catch(() => {})
  }, [])

  const loadTxs = useCallback(() => {
    api.get('/transactions', { params: { month } }).then(r => setTxs(r.data.transactions || []))
  }, [month])

  const loadBudgets = useCallback(() => {
    api.get('/budgets', { params: { month } }).then(r => setBudgets(r.data.budgets || []))
  }, [month])

  useEffect(() => {
    loadTxs()
    loadBudgets()
    loadWallets()
    api.get('/categories').then(r => setCats(r.data.categories || []))
    api.get('/debts').then(r => setDebts((r.data.debts || []).filter(d => !d.is_paid)))
    api.get('/savings').then(r => setSavings(r.data.savings || []))
  }, [loadTxs, loadBudgets, loadWallets])

  usePageFocus(() => { loadTxs(); loadBudgets(); loadWallets() })

  // ── Kategori dinamis berdasarkan domain ──────────────────────
  const expenseCats = cats.filter(c => c.type === 'expense' || c.type === 'both')
  const incomeCats  = cats.filter(c => c.type === 'income'  || c.type === 'both')

  const getCatsForDomain = (domain) => {
    if (domain === 'income')  return incomeCats
    if (domain === 'expense') return expenseCats
    return []
  }

  // ── Buka form ─────────────────────────────────────────────────
  const openCreate = () => {
    setForm(EMPTY_FORM)
    setEditTx(null)
    setShowForm(true)
  }

  const openEdit = (tx) => {
    // Edit hanya untuk income/expense manual
    setForm({
      domain: tx.type, // 'income' | 'expense'
      amount: tx.amount,
      category_name: tx.category_name,
      date: tx.date?.substring(0, 10) || today(),
      note: tx.note || '',
      debt_id: '', ef_direction: 'setor', saving_id: '',
      wallet_id: tx.wallet_id || '',
    })
    setEditTx(tx)
    setShowForm(true)
  }

  // ── Submit berdasarkan domain ─────────────────────────────────
  const handleSubmit = async (e) => {
    e.preventDefault()
    setLoading(true)
    try {
      const { domain, amount, category_name, date, note,
              debt_id, ef_direction, saving_id, wallet_id } = form
      const amt = parseFloat(amount)

      if (editTx) {
        // Edit hanya income/expense
        await api.put(`/transactions/${editTx.id}`, {
          type: domain, amount: amt, category_name, member: editTx.member, date, note,
          wallet_id: wallet_id || undefined,
        })

      } else if (domain === 'income' || domain === 'expense') {
        await api.post('/transactions', {
          type: domain, amount: amt, category_name, member: username, date, note,
          wallet_id: wallet_id || undefined,
        })

      } else if (domain === 'hutang') {
        await api.post(`/debts/${debt_id}/pay`, {
          paid_by: username, amount: amt, date,
        })
        // Reload debts supaya sisa hutang ter-update
        api.get('/debts').then(r => setDebts((r.data.debts || []).filter(d => !d.is_paid)))

      } else if (domain === 'dana-darurat') {
        const signedAmt = ef_direction === 'tarik' ? -Math.abs(amt) : Math.abs(amt)
        await api.post('/budgets/emergency/deposit', {
          amount: signedAmt, note, date,
        })

      } else if (domain === 'tabungan') {
        await api.post(`/savings/${saving_id}/deposit`, {
          amount: amt, date, note, member: username,
        })
        api.get('/savings').then(r => setSavings(r.data.savings || []))
      }

      setShowForm(false)
      loadTxs()
      loadWallets()
    } catch (err) {
      alert(err.response?.data?.error || 'Gagal menyimpan transaksi')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (id) => {
    if (!confirm('Hapus transaksi ini?')) return
    await api.delete(`/transactions/${id}`).catch(e => alert(e.response?.data?.error))
    loadTxs()
    loadWallets()
  }

  // ── Badge domain untuk baris tabel ───────────────────────────
  const getDomainBadge = (tx) => {
    if (tx.debt_payment_id)   return { label: '🏦 Cicilan Hutang', cls: 'bg-purple-100 text-purple-700' }
    if (tx.saving_deposit_id) return { label: '🎯 Tabungan',       cls: 'bg-teal-100 text-teal-700' }
    if (tx.type === 'income') return { label: '↑ Pemasukan',       cls: 'bg-green-100 text-green-700' }
    return                           { label: '↓ Pengeluaran',     cls: 'bg-red-100 text-red-600' }
  }

  // Cari dompet berdasarkan wallet_id dari transaksi
  const walletMap = Object.fromEntries(wallets.map(w => [w.id, w]))

  // ── Kategori / pilihan untuk form domain saat ini ─────────────
  const domainCats = getCatsForDomain(form.domain)

  // label tombol submit
  const submitLabel = () => {
    if (loading) return 'Menyimpan...'
    const d = form.domain
    if (d === 'hutang')       return 'Bayar Cicilan'
    if (d === 'dana-darurat') return form.ef_direction === 'tarik' ? 'Tarik Dana' : 'Setor Dana'
    if (d === 'tabungan')     return 'Setor Tabungan'
    return 'Simpan'
  }

  // ── Hitung spending dari transaksi yang sudah di-load ─────────
  const spending = {}
  for (const t of txs) {
    if (t.type === 'expense') spending[t.category_name] = (spending[t.category_name] || 0) + t.amount
  }

  // Budget rows yang punya data (merge: kategori ada anggaran ATAU ada pengeluaran)
  const budgetCatNames = new Set(budgets.map(b => b.category_name))
  const spendingCatNames = Object.keys(spending)
  const allBudgetCategories = [
    // kategori yang punya anggaran (dengan atau tanpa spending)
    ...budgets.map(b => ({
      cat: b.category_name,
      budget: b.amount,
      spent: spending[b.category_name] || 0,
    })),
    // kategori yang punya spending tapi TIDAK punya anggaran (over, tanpa set)
    ...spendingCatNames
      .filter(c => !budgetCatNames.has(c))
      .map(c => ({ cat: c, budget: 0, spent: spending[c] })),
  ].sort((a, b) => b.spent - a.spent)

  const totalBudget  = budgets.reduce((s, b) => s + b.amount, 0)
  const totalSpent   = allBudgetCategories.reduce((s, r) => s + r.spent, 0)
  const overCount    = allBudgetCategories.filter(r => r.budget > 0 && r.spent > r.budget).length

  return (
    <div className="p-4 sm:p-6 space-y-4">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <h1 className="text-xl font-bold text-gray-800">💳 Transaksi</h1>
        <div className="flex gap-2 items-center flex-wrap">
          <input
            type="month"
            value={month}
            onChange={e => setMonth(e.target.value)}
            className="input w-40"
          />
          <button onClick={openCreate} className="btn-primary">+ Tambah</button>
        </div>
      </div>

      {/* ── Panel perbandingan anggaran vs pengeluaran ── */}
      <div className="card p-0 overflow-hidden">
        {/* Header panel — klik untuk expand/collapse */}
        <button
          type="button"
          onClick={() => setBudgetOpen(o => !o)}
          className="w-full flex items-center justify-between px-4 py-3 hover:bg-gray-50 transition-colors"
        >
          <div className="flex items-center gap-3">
            <span className="text-sm font-semibold text-gray-700">📊 Pengeluaran vs Anggaran</span>
            {overCount > 0 && (
              <span className="text-xs bg-red-100 text-red-600 px-2 py-0.5 rounded-full font-medium">
                {overCount} over
              </span>
            )}
          </div>
          <div className="flex items-center gap-4">
            {totalBudget > 0 && (
              <div className="hidden sm:flex gap-4 text-xs text-gray-500">
                <span>Anggaran: <strong className="text-gray-700">{rupiah(totalBudget)}</strong></span>
                <span>Terpakai: <strong className={totalSpent > totalBudget ? 'text-red-600' : 'text-gray-700'}>{rupiah(totalSpent)}</strong></span>
                <span>Sisa: <strong className={totalBudget - totalSpent >= 0 ? 'text-green-600' : 'text-red-600'}>{rupiah(totalBudget - totalSpent)}</strong></span>
              </div>
            )}
            <span className="text-gray-400 text-xs">{budgetOpen ? '▲' : '▼'}</span>
          </div>
        </button>

        {budgetOpen && (
          <div className="border-t border-gray-100">
            {allBudgetCategories.length === 0 ? (
              <p className="px-4 py-4 text-xs text-gray-400 italic">
                Belum ada transaksi pengeluaran atau anggaran di bulan ini.
              </p>
            ) : (
              <div className="divide-y divide-gray-50">
                {allBudgetCategories.map(({ cat, budget, spent }) => {
                  const pct  = budget > 0 ? Math.min(100, (spent / budget) * 100) : 0
                  const over = budget > 0 && spent > budget
                  const noBudget = budget === 0

                  return (
                    <div key={cat} className="px-4 py-2.5 hover:bg-gray-50">
                      <div className="flex items-center justify-between mb-1.5">
                        <div className="flex items-center gap-2">
                          <span className="text-xs font-medium text-gray-700">{cat}</span>
                          {over && (
                            <span className="text-[10px] bg-red-100 text-red-600 px-1.5 py-0.5 rounded-full font-medium">Over!</span>
                          )}
                          {noBudget && (
                            <span className="text-[10px] bg-gray-100 text-gray-400 px-1.5 py-0.5 rounded-full">Tanpa anggaran</span>
                          )}
                        </div>
                        <div className="flex items-center gap-3 text-xs">
                          <span className="text-red-500 font-medium">{rupiah(spent)}</span>
                          {budget > 0 && (
                            <>
                              <span className="text-gray-300">/</span>
                              <span className="text-gray-500">{rupiah(budget)}</span>
                              <span className={`font-medium min-w-[3rem] text-right ${over ? 'text-red-600' : 'text-green-600'}`}>
                                {over ? `-${rupiah(spent - budget)}` : rupiah(budget - spent)}
                              </span>
                            </>
                          )}
                        </div>
                      </div>
                      {budget > 0 && (
                        <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
                          <div
                            className={`h-full rounded-full transition-all ${progressColor(pct)}`}
                            style={{ width: `${pct}%` }}
                          />
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Tabel transaksi */}
      <div className="card overflow-hidden p-0 overflow-x-auto">
        <table className="w-full text-sm min-w-[600px]">
          <thead>
            <tr className="bg-gray-50 border-b border-gray-100">
              <th className="text-left px-4 py-3 font-medium text-gray-600">Tanggal</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">Jenis</th>
              <th className="text-right px-4 py-3 font-medium text-gray-600">Jumlah</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">Kategori</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">Dompet</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">Anggota</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">Catatan</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody>
            {txs.length === 0 && (
              <tr><td colSpan={8} className="text-center py-8 text-gray-400">Belum ada transaksi bulan ini</td></tr>
            )}
            {txs.map(tx => {
              const badge     = getDomainBadge(tx)
              const isAuto    = !!(tx.debt_payment_id || tx.saving_deposit_id)
              const wallet    = tx.wallet_id ? walletMap[tx.wallet_id] : null
              return (
                <tr key={tx.id} className={`border-b border-gray-50 hover:bg-gray-50 ${tx.type === 'income' ? 'bg-green-50/30' : ''}`}>
                  <td className="px-4 py-2.5 text-gray-500">{fmtDate(tx.date)}</td>
                  <td className="px-4 py-2.5">
                    <span className={`badge ${badge.cls}`}>{badge.label}</span>
                  </td>
                  <td className={`px-4 py-2.5 text-right font-medium ${tx.type === 'income' ? 'text-green-600' : 'text-red-500'}`}>
                    {rupiah(tx.amount)}
                  </td>
                  <td className="px-4 py-2.5 text-gray-600">{tx.category_name}</td>
                  <td className="px-4 py-2.5">
                    {wallet ? (
                      <span className="flex items-center gap-1 text-xs">
                        <span className="w-2 h-2 rounded-full inline-block" style={{ background: wallet.color || '#6366f1' }} />
                        <span className="text-gray-600">{wallet.name}</span>
                      </span>
                    ) : <span className="text-gray-300 text-xs">—</span>}
                  </td>
                  <td className="px-4 py-2.5 text-gray-600">{tx.member}</td>
                  <td className="px-4 py-2.5 text-gray-400 max-w-xs truncate">{tx.note || '-'}</td>
                  <td className="px-4 py-2.5">
                    <div className="flex gap-1 justify-end">
                      {isAuto ? (
                        <span className="text-xs text-gray-300 px-2 py-1 italic"
                          title={tx.debt_payment_id ? 'Dari pembayaran cicilan hutang' : 'Dari setoran tabungan'}>
                          otomatis
                        </span>
                      ) : (
                        <button onClick={() => openEdit(tx)} className="btn-ghost text-xs px-2 py-1">✏️</button>
                      )}
                      <button onClick={() => handleDelete(tx.id)} className="btn-ghost text-xs px-2 py-1 text-red-500">🗑</button>
                    </div>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {/* Modal Tambah / Edit */}
      {showForm && (
        <Modal
          title={editTx ? 'Edit Transaksi' : 'Tambah Transaksi'}
          onClose={() => setShowForm(false)}
        >
          <form onSubmit={handleSubmit} className="space-y-3">

            {/* ── Pilih domain (hanya saat create) ── */}
            {!editTx && (
              <div>
                <label className="label">Jenis Transaksi</label>
                <div className="grid grid-cols-2 gap-2">
                  {DOMAINS.map(d => (
                    <button
                      key={d.value}
                      type="button"
                      onClick={() => setForm(f => ({ ...f, domain: d.value, category_name: '', debt_id: '', saving_id: '' }))}
                      className={`px-3 py-2 rounded-lg text-xs font-medium border transition-all text-left
                        ${form.domain === d.value
                          ? `${d.color} border-current ring-1 ring-current`
                          : 'bg-gray-50 border-gray-200 text-gray-600 hover:bg-gray-100'
                        }`}
                    >
                      {d.label}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* ── Jumlah (semua domain) ── */}
            <div>
              <label className="label">Jumlah (Rp)</label>
              <input className="input" type="number" min="1" step="any"
                value={form.amount}
                onChange={e => setForm(f => ({ ...f, amount: e.target.value }))}
                required />
            </div>

            {/* ── Kategori — income / expense ── */}
            {(form.domain === 'income' || form.domain === 'expense' || editTx) && (
              <div>
                <label className="label">Kategori</label>
                <select className="input" value={form.category_name}
                  onChange={e => setForm(f => ({ ...f, category_name: e.target.value }))} required>
                  <option value="">-- Pilih Kategori --</option>
                  {domainCats.map(c => <option key={c.name} value={c.name}>{c.name}</option>)}
                </select>
              </div>
            )}

            {/* ── Pilih hutang ── */}
            {form.domain === 'hutang' && !editTx && (
              <div>
                <label className="label">Hutang yang Dibayar</label>
                {debts.length === 0
                  ? <p className="text-xs text-gray-400 italic">Tidak ada hutang aktif</p>
                  : (
                    <select className="input" value={form.debt_id}
                      onChange={e => {
                        const d = debts.find(x => x.id === e.target.value)
                        setForm(f => ({ ...f, debt_id: e.target.value, amount: d ? d.monthly_payment : f.amount }))
                      }} required>
                      <option value="">-- Pilih Hutang --</option>
                      {debts.map(d => (
                        <option key={d.id} value={d.id}>
                          {d.name} — sisa {rupiah(d.remaining_amount)}
                        </option>
                      ))}
                    </select>
                  )
                }
              </div>
            )}

            {/* ── Arah dana darurat ── */}
            {form.domain === 'dana-darurat' && !editTx && (
              <div>
                <label className="label">Jenis</label>
                <div className="flex gap-2">
                  {['setor', 'tarik'].map(dir => (
                    <button key={dir} type="button"
                      onClick={() => setForm(f => ({ ...f, ef_direction: dir }))}
                      className={`flex-1 py-2 rounded-lg text-sm font-medium border transition-all
                        ${form.ef_direction === dir
                          ? dir === 'setor' ? 'bg-green-100 text-green-700 border-green-300' : 'bg-red-100 text-red-600 border-red-300'
                          : 'bg-gray-50 border-gray-200 text-gray-500 hover:bg-gray-100'
                        }`}>
                      {dir === 'setor' ? '⬆ Setor' : '⬇ Tarik'}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* ── Pilih tujuan tabungan ── */}
            {form.domain === 'tabungan' && !editTx && (
              <div>
                <label className="label">Tujuan Tabungan</label>
                {savings.length === 0
                  ? <p className="text-xs text-gray-400 italic">Belum ada tujuan tabungan</p>
                  : (
                    <select className="input" value={form.saving_id}
                      onChange={e => setForm(f => ({ ...f, saving_id: e.target.value }))} required>
                      <option value="">-- Pilih Tabungan --</option>
                      {savings.map(s => (
                        <option key={s.id} value={s.id}>
                          {s.name} — {(s.progress_pct || 0).toFixed(0)}% ({rupiah(s.current_balance)} / {rupiah(s.target_amount)})
                        </option>
                      ))}
                    </select>
                  )
                }
              </div>
            )}

            {/* ── Tanggal ── */}
            <div>
              <label className="label">Tanggal</label>
              <input className="input" type="date" value={form.date}
                onChange={e => setForm(f => ({ ...f, date: e.target.value }))} required />
            </div>

            {/* ── Catatan ── */}
            <div>
              <label className="label">Catatan</label>
              <input className="input" type="text" placeholder="Opsional" value={form.note}
                onChange={e => setForm(f => ({ ...f, note: e.target.value }))} />
            </div>

            {/* ── Dompet (income/expense saja) ── */}
            {(form.domain === 'income' || form.domain === 'expense' || editTx) && wallets.length > 0 && (
              <div>
                <label className="label">Dompet <span className="text-gray-400 font-normal">(opsional)</span></label>
                <select className="input" value={form.wallet_id}
                  onChange={e => setForm(f => ({ ...f, wallet_id: e.target.value }))}>
                  <option value="">— Tanpa dompet —</option>
                  {wallets.map(w => (
                    <option key={w.id} value={w.id}>
                      {w.name} (saldo: {rupiah(w.balance)})
                    </option>
                  ))}
                </select>
                <p className="text-xs text-gray-400 mt-1">Saldo dompet akan otomatis ter-update</p>
              </div>
            )}

            {/* ── Info kontekstual ── */}
            {form.domain === 'hutang' && !editTx && (
              <p className="text-xs text-purple-600 bg-purple-50 rounded-lg px-3 py-2">
                💡 Pembayaran akan tercatat otomatis sebagai pengeluaran "Cicilan Hutang" dan mengurangi sisa hutang.
              </p>
            )}
            {form.domain === 'tabungan' && !editTx && (
              <p className="text-xs text-teal-700 bg-teal-50 rounded-lg px-3 py-2">
                💡 Setoran akan tercatat sebagai pengeluaran dengan nama tujuan tabungan dan menambah saldo tabungan.
              </p>
            )}
            {form.domain === 'dana-darurat' && !editTx && (
              <p className="text-xs text-blue-700 bg-blue-50 rounded-lg px-3 py-2">
                💡 {form.ef_direction === 'setor' ? 'Menambah saldo dana darurat.' : 'Mengurangi saldo dana darurat.'}
              </p>
            )}

            <div className="flex gap-2 pt-1">
              <button type="submit" disabled={loading} className="btn-primary flex-1">
                {submitLabel()}
              </button>
              <button type="button" onClick={() => setShowForm(false)} className="btn-outline flex-1">Batal</button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  )
}
