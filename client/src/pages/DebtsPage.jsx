import { useState } from 'react'
import api from '../api.js'
import { rupiah, fmtDate, today, pct } from '../utils.js'
import Modal from '../components/Modal.jsx'
import { usePageFocus } from '../hooks/usePageFocus.js'
import { useCurrentUser } from '../App.jsx'

const EMPTY_DEBT = { name: '', total_amount: '', monthly_payment: '', due_date: '' }

export default function DebtsPage() {
  const currentUser = useCurrentUser()
  const username = currentUser?.username || ''

  const [debts, setDebts] = useState([])
  const [showCreate, setShowCreate] = useState(false)
  const [editDebt, setEditDebt] = useState(null)
  const [payDebt, setPayDebt] = useState(null)
  const [form, setForm] = useState(EMPTY_DEBT)
  const [payForm, setPayForm] = useState({ paid_by: username, amount: '', date: today() })
  const [loading, setLoading] = useState(false)
  // expandedPayments: { [debtId]: DebtPayment[] | null }
  const [expandedPayments, setExpandedPayments] = useState({})
  const [loadingPayments, setLoadingPayments] = useState({})

  const load = () => api.get('/debts').then(r => setDebts(r.data.debts || []))
  usePageFocus(load)

  const openCreate = () => { setForm(EMPTY_DEBT); setEditDebt(null); setShowCreate(true) }
  const openEdit = (d) => {
    setForm({ name: d.name, total_amount: d.total_amount, monthly_payment: d.monthly_payment, due_date: d.due_date?.substring(0, 10) || '' })
    setEditDebt(d)
    setShowCreate(true)
  }

  const togglePayments = async (d) => {
    const id = d.id
    // Jika sudah terbuka, tutup
    if (expandedPayments[id] !== undefined) {
      setExpandedPayments(prev => {
        const next = { ...prev }
        delete next[id]
        return next
      })
      return
    }
    // Muat riwayat
    setLoadingPayments(prev => ({ ...prev, [id]: true }))
    try {
      const res = await api.get(`/debts/${id}/payments`)
      setExpandedPayments(prev => ({ ...prev, [id]: res.data.payments || [] }))
    } catch {
      setExpandedPayments(prev => ({ ...prev, [id]: [] }))
    } finally {
      setLoadingPayments(prev => {
        const next = { ...prev }
        delete next[id]
        return next
      })
    }
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setLoading(true)
    try {
      const payload = {
        ...form,
        total_amount: parseFloat(form.total_amount),
        monthly_payment: parseFloat(form.monthly_payment),
      }
      if (editDebt) {
        await api.put(`/debts/${editDebt.id}`, payload)
      } else {
        await api.post('/debts', payload)
      }
      setShowCreate(false)
      load()
    } catch (err) {
      alert(err.response?.data?.error || 'Gagal menyimpan hutang')
    } finally {
      setLoading(false)
    }
  }

  const handlePay = async (e) => {
    e.preventDefault()
    setLoading(true)
    try {
      await api.post(`/debts/${payDebt.id}/pay`, {
        ...payForm,
        amount: parseFloat(payForm.amount),
      })
      setPayDebt(null)
      // Refresh daftar hutang
      load()
      // Refresh riwayat pembayaran jika sedang ditampilkan
      if (expandedPayments[payDebt.id] !== undefined) {
        const res = await api.get(`/debts/${payDebt.id}/payments`)
        setExpandedPayments(prev => ({ ...prev, [payDebt.id]: res.data.payments || [] }))
      }
    } catch (err) {
      alert(err.response?.data?.error || 'Gagal bayar cicilan')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (id) => {
    if (!confirm('Hapus hutang ini beserta semua riwayat pembayaran?')) return
    await api.delete(`/debts/${id}`).catch(e => alert(e.response?.data?.error))
    setExpandedPayments(prev => {
      const next = { ...prev }
      delete next[id]
      return next
    })
    load()
  }

  return (
    <div className="p-6 space-y-5">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-gray-800">🏦 Hutang</h1>
        <button onClick={openCreate} className="btn-primary">+ Tambah Hutang</button>
      </div>

      {debts.length === 0 && (
        <div className="card text-center text-gray-400 py-10">Belum ada hutang yang dicatat</div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {debts.map(d => {
          const progress = pct(d.total_amount - d.remaining_amount, d.total_amount)
          const isUrgent = !d.is_paid && d.days_until_due <= 7
          const payments = expandedPayments[d.id]
          const isExpanded = payments !== undefined
          const isLoadingPmts = !!loadingPayments[d.id]

          return (
            <div key={d.id} className={`card ${d.is_paid ? 'opacity-60' : ''}`}>
              <div className="flex items-start justify-between mb-2">
                <div>
                  <h3 className="font-semibold text-gray-800">{d.name}</h3>
                  <div className="text-xs text-gray-400">Total: {rupiah(d.total_amount)}</div>
                </div>
                <div className="flex gap-1">
                  {d.is_paid
                    ? <span className="badge bg-green-100 text-green-700">✅ Lunas</span>
                    : isUrgent
                      ? <span className="badge bg-red-100 text-red-700">⚠️ Segera!</span>
                      : null
                  }
                </div>
              </div>

              {/* Progress bar */}
              <div className="mb-3">
                <div className="flex justify-between text-xs text-gray-500 mb-1">
                  <span>Terbayar: {rupiah(d.total_amount - d.remaining_amount)}</span>
                  <span>Sisa: {rupiah(d.remaining_amount)}</span>
                </div>
                <div className="h-2.5 bg-gray-100 rounded-full overflow-hidden">
                  <div className="h-full bg-blue-500 rounded-full" style={{ width: `${progress}%` }} />
                </div>
                <div className="text-xs text-gray-400 mt-0.5 text-right">{progress.toFixed(0)}% lunas</div>
              </div>

              <div className="flex items-center justify-between text-xs text-gray-500">
                <span>Cicilan: {rupiah(d.monthly_payment)}/bln</span>
                <span className={isUrgent ? 'text-red-600 font-medium' : ''}>
                  Jatuh tempo: {fmtDate(d.due_date)}
                  {!d.is_paid && ` (${d.days_until_due > 0 ? `${d.days_until_due} hari` : 'Lewat!'})`}
                </span>
              </div>

              {/* Aksi */}
              <div className="flex gap-2 mt-3 pt-3 border-t border-gray-100">
                {/* Tombol riwayat pembayaran */}
                <button
                  onClick={() => togglePayments(d)}
                  className="btn-outline text-xs py-1.5 px-3 flex items-center gap-1"
                  disabled={isLoadingPmts}
                >
                  {isLoadingPmts ? '⏳' : isExpanded ? '▲' : '📋'}
                  {isLoadingPmts ? 'Memuat...' : isExpanded ? 'Tutup' : 'Riwayat'}
                </button>

                {!d.is_paid && (
                  <>
                    <button
                      onClick={() => { setPayDebt(d); setPayForm({ paid_by: username, amount: d.monthly_payment, date: today() }) }}
                      className="btn-primary flex-1 text-xs py-1.5"
                    >
                      💳 Bayar Cicilan
                    </button>
                    <button onClick={() => openEdit(d)} className="btn-outline text-xs py-1.5 px-3">✏️</button>
                  </>
                )}
                <button onClick={() => handleDelete(d.id)} className="btn-outline text-xs py-1.5 px-3 text-red-500">🗑</button>
              </div>

              {/* Riwayat Pembayaran — expandable */}
              {isExpanded && (
                <div className="mt-3 border-t border-gray-100 pt-3">
                  <p className="text-xs font-semibold text-gray-600 mb-2">
                    📋 Riwayat Pembayaran
                    <span className="ml-1 text-gray-400 font-normal">({payments.length} transaksi)</span>
                  </p>
                  {payments.length === 0 ? (
                    <p className="text-xs text-gray-400 italic">Belum ada pembayaran tercatat</p>
                  ) : (
                    <div className="space-y-1.5 max-h-48 overflow-y-auto pr-1">
                      {payments.map(p => (
                        <div key={p.id} className="flex items-center justify-between text-xs bg-gray-50 rounded px-2.5 py-1.5">
                          <div className="flex items-center gap-2">
                            <span className="text-gray-400">{fmtDate(p.date)}</span>
                            <span className="badge bg-blue-50 text-blue-600 py-0">{p.paid_by}</span>
                          </div>
                          <span className="font-medium text-red-500">{rupiah(p.amount)}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          )
        })}
      </div>

      {/* Modal Tambah/Edit Hutang */}
      {showCreate && (
        <Modal title={editDebt ? 'Edit Hutang' : 'Tambah Hutang'} onClose={() => setShowCreate(false)}>
          <form onSubmit={handleSubmit} className="space-y-3">
            <div>
              <label className="label">Nama Hutang</label>
              <input className="input" placeholder="cth: KPR, Cicilan Motor" value={form.name}
                onChange={e => setForm(f => ({ ...f, name: e.target.value }))} required />
            </div>
            <div>
              <label className="label">Total Hutang (Rp)</label>
              <input className="input" type="number" min="1" step="any" value={form.total_amount}
                onChange={e => setForm(f => ({ ...f, total_amount: e.target.value }))} required />
            </div>
            <div>
              <label className="label">Cicilan per Bulan (Rp)</label>
              <input className="input" type="number" min="1" step="any" value={form.monthly_payment}
                onChange={e => setForm(f => ({ ...f, monthly_payment: e.target.value }))} required />
            </div>
            <div>
              <label className="label">Tanggal Jatuh Tempo</label>
              <input className="input" type="date" value={form.due_date}
                onChange={e => setForm(f => ({ ...f, due_date: e.target.value }))} required />
            </div>
            <div className="flex gap-2 pt-1">
              <button type="submit" disabled={loading} className="btn-primary flex-1">
                {loading ? 'Menyimpan...' : 'Simpan'}
              </button>
              <button type="button" onClick={() => setShowCreate(false)} className="btn-outline flex-1">Batal</button>
            </div>
          </form>
        </Modal>
      )}

      {/* Modal Bayar Cicilan */}
      {payDebt && (
        <Modal title={`Bayar Cicilan — ${payDebt.name}`} onClose={() => setPayDebt(null)}>
          <form onSubmit={handlePay} className="space-y-3">
            <div>
              <label className="label">Dibayar oleh</label>
              <input
                className="input bg-gray-50"
                type="text"
                value={username}
                readOnly
                title="Otomatis diisi dengan username Anda"
              />
            </div>
            <div>
              <label className="label">Jumlah Bayar (Rp)</label>
              <input className="input" type="number" min="1" step="any" value={payForm.amount}
                onChange={e => setPayForm(f => ({ ...f, amount: e.target.value }))} required />
            </div>
            <div>
              <label className="label">Tanggal Bayar</label>
              <input className="input" type="date" value={payForm.date}
                onChange={e => setPayForm(f => ({ ...f, date: e.target.value }))} required />
            </div>
            <p className="text-xs text-gray-400">Pembayaran akan otomatis tercatat sebagai pengeluaran "Cicilan Hutang" di halaman Transaksi</p>
            <div className="flex gap-2 pt-1">
              <button type="submit" disabled={loading} className="btn-primary flex-1">
                {loading ? 'Memproses...' : 'Bayar'}
              </button>
              <button type="button" onClick={() => setPayDebt(null)} className="btn-outline flex-1">Batal</button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  )
}
