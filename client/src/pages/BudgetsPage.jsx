import { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import api from '../api.js'
import { usePageFocus } from '../hooks/usePageFocus.js'
import { rupiah, thisMonth, progressColor } from '../utils.js'
import Modal from '../components/Modal.jsx'

// Mengembalikan YYYY-MM bulan sebelumnya dari bulan yang diberikan
function prevMonth(ym) {
  const [y, m] = ym.split('-').map(Number)
  const d = new Date(y, m - 2, 1) // m-2 karena Date bulan 0-based
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

export default function BudgetsPage() {
  const [budgets, setBudgets] = useState([])
  const [cats, setCats] = useState([])
  const [ef, setEf] = useState(null)
  const [month, setMonth] = useState(thisMonth())
  const [showForm, setShowForm] = useState(false)
  const [editBudget, setEditBudget] = useState(null)
  const [form, setForm] = useState({ category_name: '', month: thisMonth(), amount: '' })
  const [spending, setSpending] = useState({})
  const [loading, setLoading] = useState(false)
  const [copyLoading, setCopyLoading] = useState(false)
  const [debts, setDebts] = useState([])
  const [debtLoading, setDebtLoading] = useState(false)
  // Estimasi pendapatan: rata-rata pemasukan 3 bulan sebelumnya + realisasi bulan terpilih
  const [incomeEst, setIncomeEst] = useState({ avg: 0, monthsWithData: 0, actual: 0 })

  // ── Recurring budget state ────────────────────────────────
  const [recurring, setRecurring] = useState([])
  const [showRecurringForm, setShowRecurringForm] = useState(false)
  const [recurringForm, setRecurringForm] = useState({ category_name: '', amount: '' })
  const [recurringLoading, setRecurringLoading] = useState(false)
  const [applyLoading, setApplyLoading] = useState(false)

  const loadAll = useCallback(() => {
    api.get('/budgets', { params: { month } }).then(r => setBudgets(r.data.budgets || []))
    api.get('/budgets/emergency').then(r => setEf(r.data.emergency_fund))
    api.get('/budgets/recurring').then(r => setRecurring(r.data.recurring_budgets || []))
    api.get('/categories').then(r => setCats(r.data.categories || []))
    api.get('/debts').then(r => setDebts(r.data.debts || [])).catch(() => setDebts([]))
    // Ambil pengeluaran bulan ini per kategori dari transactions
    api.get('/transactions', { params: { month } }).then(r => {
      const sp = {}
      let actualIncome = 0
      for (const t of (r.data.transactions || [])) {
        if (t.type === 'expense') sp[t.category_name] = (sp[t.category_name] || 0) + t.amount
        else if (t.type === 'income') actualIncome += t.amount
      }
      setSpending(sp)
      setIncomeEst(e => ({ ...e, actual: actualIncome }))
    })
    // Pemasukan 3 bulan sebelumnya untuk estimasi pendapatan
    const prevMonths = []
    for (let i = 0, m = month; i < 3; i++) { m = prevMonth(m); prevMonths.push(m) }
    Promise.all(prevMonths.map(m =>
      api.get('/transactions', { params: { month: m } })
        .then(r => (r.data.transactions || []).reduce((s, t) => s + (t.type === 'income' ? t.amount : 0), 0))
        .catch(() => 0)
    )).then(totals => {
      const withData = totals.filter(t => t > 0)
      const avg = withData.length ? withData.reduce((s, t) => s + t, 0) / withData.length : 0
      setIncomeEst(e => ({ ...e, avg, monthsWithData: withData.length }))
    })
  }, [month])

  useEffect(loadAll, [loadAll])
  usePageFocus(loadAll)

  const openCreate = () => {
    setForm({ category_name: '', month, amount: '' })
    setEditBudget(null)
    setShowForm(true)
  }

  const openEdit = (b) => {
    setForm({ category_name: b.category_name, month: b.month, amount: b.amount })
    setEditBudget(b)
    setShowForm(true)
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setLoading(true)
    try {
      const payload = { ...form, amount: parseFloat(form.amount) }
      if (editBudget) {
        await api.put(`/budgets/${editBudget.id}`, { amount: payload.amount })
      } else {
        await api.post('/budgets', payload)
      }
      setShowForm(false)
      loadAll()
    } catch (err) {
      alert(err.response?.data?.error || 'Gagal menyimpan anggaran')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (id) => {
    if (!confirm('Hapus anggaran ini?')) return
    await api.delete(`/budgets/${id}`).catch(e => alert(e.response?.data?.error))
    loadAll()
  }

  // Salin semua anggaran dari bulan sebelumnya ke bulan aktif
  const handleCopyPrev = async () => {
    const src = prevMonth(month)
    const r = await api.get('/budgets', { params: { month: src } }).catch(() => null)
    const srcBudgets = r?.data?.budgets || []
    if (srcBudgets.length === 0) {
      alert(`Tidak ada anggaran di bulan ${src} untuk disalin.`)
      return
    }
    // Kategori yang sudah ada di bulan aktif — jangan timpa
    const existing = new Set(budgets.map(b => b.category_name))
    const toAdd = srcBudgets.filter(b => !existing.has(b.category_name))
    if (toAdd.length === 0) {
      alert('Semua kategori dari bulan lalu sudah ada di bulan ini.')
      return
    }
    if (!confirm(`Salin ${toAdd.length} anggaran dari ${src} ke ${month}?`)) return
    setCopyLoading(true)
    try {
      await Promise.all(toAdd.map(b =>
        api.post('/budgets', { category_name: b.category_name, month, amount: b.amount })
      ))
      loadAll()
    } catch (e) {
      alert(e.response?.data?.error || 'Gagal menyalin anggaran')
    } finally {
      setCopyLoading(false)
    }
  }

  // ── Recurring budget handlers ─────────────────────────────
  const openRecurringForm = () => {
    setRecurringForm({ category_name: '', amount: '' })
    setShowRecurringForm(true)
  }

  const handleRecurringSubmit = async (e) => {
    e.preventDefault()
    setRecurringLoading(true)
    try {
      await api.post('/budgets/recurring', {
        category_name: recurringForm.category_name,
        amount: parseFloat(recurringForm.amount),
      })
      setShowRecurringForm(false)
      loadAll()
    } catch (err) {
      alert(err.response?.data?.error || 'Gagal menyimpan anggaran berulang')
    } finally {
      setRecurringLoading(false)
    }
  }

  const handleRecurringDelete = async (id) => {
    if (!confirm('Hapus template anggaran berulang ini?')) return
    await api.delete(`/budgets/recurring/${id}`).catch(e => alert(e.response?.data?.error))
    loadAll()
  }

  const handleApplyRecurring = async () => {
    if (recurring.length === 0) {
      alert('Belum ada template anggaran berulang.')
      return
    }
    if (!confirm(`Terapkan ${recurring.length} anggaran berulang ke bulan ${month}?\n(Anggaran yang sudah ada tidak akan ditimpa)`)) return
    setApplyLoading(true)
    try {
      const r = await api.post('/budgets/recurring/apply', { month })
      const count = r.data.applied_count ?? 0
      if (count === 0) {
        alert('Semua kategori dari template sudah memiliki anggaran di bulan ini.')
      } else {
        alert(`${count} anggaran berhasil diterapkan ke ${month}.`)
      }
      loadAll()
    } catch (e) {
      alert(e.response?.data?.error || 'Gagal menerapkan anggaran berulang')
    } finally {
      setApplyLoading(false)
    }
  }

  // ── Anggaran cicilan hutang (bulan berjalan) ──────────────
  const DEBT_CATEGORY = 'Cicilan Hutang'
  const activeDebts = debts.filter(d => !d.is_paid && d.remaining_amount > 0)
  // Cicilan bulanan tidak melebihi sisa hutang
  const debtBudgetAmount = activeDebts.reduce(
    (s, d) => s + Math.min(d.monthly_payment, d.remaining_amount), 0
  )
  const existingDebtBudget = budgets.find(b => b.category_name === DEBT_CATEGORY)
  const showDebtBanner = month === thisMonth() && activeDebts.length > 0 &&
    (!existingDebtBudget || existingDebtBudget.amount !== debtBudgetAmount)

  const handleAddDebtBudget = async () => {
    const msg = existingDebtBudget
      ? `Ubah anggaran "${DEBT_CATEGORY}" bulan ${month} dari ${rupiah(existingDebtBudget.amount)} menjadi ${rupiah(debtBudgetAmount)}?`
      : `Tambah anggaran "${DEBT_CATEGORY}" ${rupiah(debtBudgetAmount)} untuk bulan ${month}?`
    if (!confirm(msg)) return
    setDebtLoading(true)
    try {
      await api.post('/budgets', { category_name: DEBT_CATEGORY, month, amount: debtBudgetAmount })
      loadAll()
    } catch (e) {
      alert(e.response?.data?.error || 'Gagal menambah anggaran hutang')
    } finally {
      setDebtLoading(false)
    }
  }

  // Ringkasan total anggaran bulan terpilih (default: bulan berjalan)
  const totalBudget = budgets.reduce((s, b) => s + b.amount, 0)
  const totalSpent = budgets.reduce((s, b) => s + (spending[b.category_name] || 0), 0)
  const totalRemaining = totalBudget - totalSpent
  const totalPct = totalBudget > 0 ? Math.min(100, (totalSpent / totalBudget) * 100) : 0

  // Estimasi pendapatan: pakai rata-rata historis; jika belum ada, pakai realisasi bulan ini
  const estIncome = incomeEst.avg > 0 ? incomeEst.avg : incomeEst.actual
  const estBalance = estIncome - totalBudget
  const budgetOfIncomePct = estIncome > 0 ? (totalBudget / estIncome) * 100 : 0

  const expenseCats = cats.filter(c => c.type === 'expense' || c.type === 'both')
  // Kategori yang belum ada di recurring (untuk form tambah)
  const availableCatsForRecurring = expenseCats.filter(
    c => !recurring.some(r => r.category_name === c.name)
  )

  return (
    <div className="p-4 sm:p-6 space-y-5">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <h1 className="text-xl font-bold text-gray-800">📋 Anggaran</h1>
        <div className="flex gap-2 items-center flex-wrap">
          <input type="month" value={month} onChange={e => setMonth(e.target.value)} className="input w-40" />
          <button onClick={handleCopyPrev} disabled={copyLoading} className="btn-outline text-sm">
            {copyLoading ? 'Menyalin...' : '📋 Salin dari Bulan Lalu'}
          </button>
          <button onClick={openCreate} className="btn-primary text-sm">+ Atur Anggaran</button>
        </div>
      </div>

      {/* Saran anggaran cicilan hutang */}
      {showDebtBanner && (
        <div className="card flex flex-col sm:flex-row sm:items-center justify-between gap-3 border border-amber-200 bg-amber-50">
          <div>
            <div className="text-sm font-semibold text-amber-800">🏦 Ada {activeDebts.length} hutang aktif bulan ini</div>
            <div className="text-xs text-amber-700 mt-0.5">
              Total cicilan bulan ini {rupiah(debtBudgetAmount)} belum {existingDebtBudget ? 'sesuai dengan' : 'masuk ke'} anggaran "{DEBT_CATEGORY}".
            </div>
          </div>
          <button onClick={handleAddDebtBudget} disabled={debtLoading} className="btn-primary text-sm whitespace-nowrap">
            {debtLoading ? 'Menyimpan...' : existingDebtBudget ? 'Perbarui Anggaran Hutang' : '+ Tambah Anggaran Hutang'}
          </button>
        </div>
      )}

      {/* Ringkasan Total Anggaran */}
      <div className="card">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-semibold text-gray-700">
            Total Anggaran {month === thisMonth() ? 'Bulan Berjalan' : 'Bulan Terpilih'} ({month})
          </h3>
          <span className="text-xs text-gray-400">{budgets.length} kategori</span>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div>
            <div className="text-xs text-gray-400">Total Anggaran</div>
            <div className="text-lg font-bold text-gray-800">{rupiah(totalBudget)}</div>
          </div>
          <div>
            <div className="text-xs text-gray-400">Total Terpakai</div>
            <div className="text-lg font-bold text-red-500">{rupiah(totalSpent)}</div>
          </div>
          <div>
            <div className="text-xs text-gray-400">Total Sisa</div>
            <div className={`text-lg font-bold ${totalRemaining >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {rupiah(totalRemaining)}
            </div>
          </div>
        </div>
        <div className="h-2 bg-gray-100 rounded-full overflow-hidden mt-3">
          <div className={`h-full rounded-full ${progressColor(totalPct)}`} style={{ width: `${totalPct}%` }} />
        </div>
        <div className="text-xs text-gray-400 mt-0.5 text-right">{totalPct.toFixed(0)}% terpakai</div>
      </div>

      {/* Estimasi Pendapatan */}
      <div className="card">
        <div className="flex items-center justify-between mb-3">
          <div>
            <h3 className="text-sm font-semibold text-gray-700">💰 Estimasi Pendapatan ({month})</h3>
            <p className="text-xs text-gray-400 mt-0.5">
              {incomeEst.avg > 0
                ? `Rata-rata pemasukan ${incomeEst.monthsWithData} bulan terakhir yang memiliki data`
                : 'Belum ada riwayat pemasukan 3 bulan terakhir — memakai realisasi bulan ini'}
            </p>
          </div>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div>
            <div className="text-xs text-gray-400">Estimasi Pendapatan</div>
            <div className="text-lg font-bold text-gray-800">{rupiah(estIncome)}</div>
          </div>
          <div>
            <div className="text-xs text-gray-400">Realisasi Bulan Ini</div>
            <div className="text-lg font-bold text-green-600">{rupiah(incomeEst.actual)}</div>
          </div>
          <div>
            <div className="text-xs text-gray-400">Estimasi Sisa setelah Anggaran</div>
            <div className={`text-lg font-bold ${estBalance >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {rupiah(estBalance)}
            </div>
          </div>
        </div>
        {estIncome > 0 && (
          <>
            <div className="h-2 bg-gray-100 rounded-full overflow-hidden mt-3">
              <div className={`h-full rounded-full ${progressColor(Math.min(100, budgetOfIncomePct))}`}
                style={{ width: `${Math.min(100, budgetOfIncomePct)}%` }} />
            </div>
            <div className="text-xs text-gray-400 mt-0.5 text-right">
              Anggaran = {budgetOfIncomePct.toFixed(0)}% dari estimasi pendapatan
            </div>
            {estBalance < 0 && (
              <div className="text-xs text-red-600 mt-1">
                ⚠ Total anggaran melebihi estimasi pendapatan sebesar {rupiah(-estBalance)}.
              </div>
            )}
          </>
        )}
      </div>

      {/* Tabel Anggaran */}
      <div className="card overflow-x-auto p-0">
        <table className="w-full text-sm min-w-[480px]">
          <thead>
            <tr className="bg-gray-50 border-b border-gray-100">
              <th className="text-left px-4 py-3 font-medium text-gray-600">Kategori</th>
              <th className="text-right px-4 py-3 font-medium text-gray-600">Anggaran</th>
              <th className="text-right px-4 py-3 font-medium text-gray-600">Terpakai</th>
              <th className="text-right px-4 py-3 font-medium text-gray-600">Sisa</th>
              <th className="px-4 py-3 w-32">Progress</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody>
            {budgets.length === 0 && (
              <tr><td colSpan={6} className="text-center py-8 text-gray-400">Belum ada anggaran untuk bulan ini</td></tr>
            )}
            {budgets.map(b => {
              const spent = spending[b.category_name] || 0
              const remaining = b.amount - spent
              const pct = b.amount > 0 ? Math.min(100, (spent / b.amount) * 100) : 0
              return (
                <tr key={b.id} className="border-b border-gray-50 hover:bg-gray-50">
                  <td className="px-4 py-2.5 font-medium">
                    {b.category_name}
                    {recurring.some(r => r.category_name === b.category_name) && (
                      <span className="ml-1.5 text-xs bg-indigo-50 text-indigo-600 px-1.5 py-0.5 rounded-full">🔁</span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 text-right">{rupiah(b.amount)}</td>
                  <td className="px-4 py-2.5 text-right text-red-500">{rupiah(spent)}</td>
                  <td className={`px-4 py-2.5 text-right font-medium ${remaining >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                    {rupiah(remaining)}
                  </td>
                  <td className="px-4 py-2.5">
                    <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
                      <div className={`h-full rounded-full ${progressColor(pct)}`} style={{ width: `${pct}%` }} />
                    </div>
                    <div className="text-xs text-gray-400 mt-0.5 text-right">{pct.toFixed(0)}%</div>
                  </td>
                  <td className="px-4 py-2.5">
                    <div className="flex gap-1 justify-end">
                      <button onClick={() => openEdit(b)} className="btn-ghost text-xs px-2 py-1">✏️</button>
                      <button onClick={() => handleDelete(b.id)} className="btn-ghost text-xs px-2 py-1 text-red-500">🗑</button>
                    </div>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {/* Anggaran Berulang */}
      <div className="card">
        <div className="flex items-center justify-between mb-3">
          <div>
            <h3 className="text-sm font-semibold text-gray-700">🔁 Anggaran Berulang</h3>
            <p className="text-xs text-gray-400 mt-0.5">Template otomatis yang bisa diterapkan ke bulan manapun</p>
          </div>
          <div className="flex gap-2">
            <button
              onClick={handleApplyRecurring}
              disabled={applyLoading || recurring.length === 0}
              className="btn-outline text-xs"
            >
              {applyLoading ? 'Menerapkan...' : `▶ Terapkan ke ${month}`}
            </button>
            <button onClick={openRecurringForm} className="btn-primary text-xs">+ Tambah Template</button>
          </div>
        </div>

        {recurring.length === 0 ? (
          <p className="text-xs text-gray-400 py-2">Belum ada template anggaran berulang. Tambahkan agar anggaran bisa diterapkan otomatis tiap bulan.</p>
        ) : (
          <div className="overflow-x-auto rounded-lg border border-gray-100">
            <table className="w-full text-sm min-w-[320px]">
              <thead>
                <tr className="bg-gray-50 border-b border-gray-100">
                  <th className="text-left px-4 py-2.5 font-medium text-gray-600">Kategori</th>
                  <th className="text-right px-4 py-2.5 font-medium text-gray-600">Jumlah / Bulan</th>
                  <th className="px-4 py-2.5"></th>
                </tr>
              </thead>
              <tbody>
                {recurring.map(rb => (
                  <tr key={rb.id} className="border-b border-gray-50 hover:bg-gray-50 last:border-0">
                    <td className="px-4 py-2 font-medium">{rb.category_name}</td>
                    <td className="px-4 py-2 text-right text-indigo-600 font-semibold">{rupiah(rb.amount)}</td>
                    <td className="px-4 py-2 text-right">
                      <button
                        onClick={() => handleRecurringDelete(rb.id)}
                        className="btn-ghost text-xs px-2 py-1 text-red-500"
                      >🗑</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Dana Darurat — shortcut ke halaman dedicated */}
      <Link to="/emergency-fund" className="card block hover:shadow-md transition-shadow">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-2xl">🛡</span>
            <div>
              <div className="text-sm font-semibold text-gray-700">Dana Darurat</div>
              <div className="text-xs text-gray-400">Lihat detail, setor & riwayat transaksi</div>
            </div>
          </div>
          <div className="text-right">
            {ef ? (
              <>
                <div className="text-sm font-bold text-blue-600">{rupiah(ef.current_balance)}</div>
                <div className={`text-xs font-medium ${(ef.months_covered || 0) >= (ef.target_months || 6) ? 'text-green-600' : 'text-red-500'}`}>
                  {(ef.months_covered || 0).toFixed(1)} / {ef.target_months || 6} bulan
                </div>
              </>
            ) : <div className="text-xs text-gray-400">Belum ada data</div>}
          </div>
        </div>
      </Link>

      {/* Modal Anggaran */}
      {showForm && (
        <Modal title={editBudget ? 'Edit Anggaran' : 'Atur Anggaran'} onClose={() => setShowForm(false)}>
          <form onSubmit={handleSubmit} className="space-y-3">
            {!editBudget && (
              <>
                <div>
                  <label className="label">Kategori</label>
                  <select className="input" value={form.category_name}
                    onChange={e => setForm(f => ({ ...f, category_name: e.target.value }))} required>
                    <option value="">-- Pilih Kategori --</option>
                    {expenseCats.map(c => <option key={c.name} value={c.name}>{c.name}</option>)}
                  </select>
                </div>
                <div>
                  <label className="label">Bulan</label>
                  <input className="input" type="month" value={form.month}
                    onChange={e => setForm(f => ({ ...f, month: e.target.value }))} required />
                </div>
              </>
            )}
            {editBudget && (
              <p className="text-sm text-gray-600">Kategori: <strong>{editBudget.category_name}</strong> ({editBudget.month})</p>
            )}
            <div>
              <label className="label">Jumlah Anggaran (Rp)</label>
              <input className="input" type="number" min="0" step="any" value={form.amount}
                onChange={e => setForm(f => ({ ...f, amount: e.target.value }))} required />
            </div>
            <div className="flex gap-2 pt-1">
              <button type="submit" disabled={loading} className="btn-primary flex-1">
                {loading ? 'Menyimpan...' : 'Simpan'}
              </button>
              <button type="button" onClick={() => setShowForm(false)} className="btn-outline flex-1">Batal</button>
            </div>
          </form>
        </Modal>
      )}

      {/* Modal Anggaran Berulang */}
      {showRecurringForm && (
        <Modal title="Tambah Template Anggaran Berulang" onClose={() => setShowRecurringForm(false)}>
          <form onSubmit={handleRecurringSubmit} className="space-y-3">
            <p className="text-xs text-gray-500">
              Template ini akan digunakan setiap bulan. Gunakan tombol <strong>Terapkan</strong> untuk menambahkan ke bulan yang dipilih.
            </p>
            <div>
              <label className="label">Kategori</label>
              <select className="input" value={recurringForm.category_name}
                onChange={e => setRecurringForm(f => ({ ...f, category_name: e.target.value }))} required>
                <option value="">-- Pilih Kategori --</option>
                {availableCatsForRecurring.map(c => <option key={c.name} value={c.name}>{c.name}</option>)}
              </select>
              {availableCatsForRecurring.length === 0 && (
                <p className="text-xs text-amber-600 mt-1">Semua kategori sudah memiliki template berulang.</p>
              )}
            </div>
            <div>
              <label className="label">Jumlah per Bulan (Rp)</label>
              <input className="input" type="number" min="0" step="any" value={recurringForm.amount}
                onChange={e => setRecurringForm(f => ({ ...f, amount: e.target.value }))} required />
            </div>
            <div className="flex gap-2 pt-1">
              <button type="submit" disabled={recurringLoading || availableCatsForRecurring.length === 0} className="btn-primary flex-1">
                {recurringLoading ? 'Menyimpan...' : 'Simpan'}
              </button>
              <button type="button" onClick={() => setShowRecurringForm(false)} className="btn-outline flex-1">Batal</button>
            </div>
          </form>
        </Modal>
      )}

    </div>
  )
}
