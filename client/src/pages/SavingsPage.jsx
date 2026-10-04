import { useState } from 'react'
import api from '../api.js'
import { rupiah, fmtDate, today } from '../utils.js'
import Modal from '../components/Modal.jsx'
import { usePageFocus } from '../hooks/usePageFocus.js'
import { useCurrentUser } from '../App.jsx'

const EMPTY_SAVING = { name: '', target_amount: '', initial_balance: '0', target_date: '' }

export default function SavingsPage() {
  const currentUser = useCurrentUser()
  const username = currentUser?.username || ''

  const [savings, setSavings] = useState([])
  const [showCreate, setShowCreate] = useState(false)
  const [editSaving, setEditSaving] = useState(null)
  const [depositSaving, setDepositSaving] = useState(null)
  const [form, setForm] = useState(EMPTY_SAVING)
  const [depForm, setDepForm] = useState({ amount: '', date: today(), note: '' })
  const [loading, setLoading] = useState(false)

  const load = () => api.get('/savings').then(r => setSavings(r.data.savings || []))
  usePageFocus(load)

  const openCreate = () => { setForm(EMPTY_SAVING); setEditSaving(null); setShowCreate(true) }
  const openEdit = (s) => {
    setForm({ name: s.name, target_amount: s.target_amount, initial_balance: s.current_balance, target_date: s.target_date?.substring(0, 10) || '' })
    setEditSaving(s)
    setShowCreate(true)
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setLoading(true)
    try {
      if (editSaving) {
        await api.put(`/savings/${editSaving.id}`, {
          name: form.name,
          target_amount: parseFloat(form.target_amount),
          target_date: form.target_date,
        })
      } else {
        await api.post('/savings', {
          ...form,
          target_amount: parseFloat(form.target_amount),
          initial_balance: parseFloat(form.initial_balance || 0),
        })
      }
      setShowCreate(false)
      load()
    } catch (err) {
      alert(err.response?.data?.error || 'Gagal menyimpan tabungan')
    } finally {
      setLoading(false)
    }
  }

  const handleDeposit = async (e) => {
    e.preventDefault()
    setLoading(true)
    try {
      await api.post(`/savings/${depositSaving.id}/deposit`, {
        ...depForm,
        amount: parseFloat(depForm.amount),
        member: username,
      })
      setDepositSaving(null)
      load()
    } catch (err) {
      alert(err.response?.data?.error || 'Gagal mencatat setoran')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (id) => {
    if (!confirm('Hapus tujuan tabungan ini?')) return
    await api.delete(`/savings/${id}`).catch(e => alert(e.response?.data?.error))
    load()
  }

  return (
    <div className="p-6 space-y-5">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-gray-800">🎯 Tabungan</h1>
        <button onClick={openCreate} className="btn-primary">+ Tujuan Baru</button>
      </div>

      {savings.length === 0 && (
        <div className="card text-center text-gray-400 py-10">Belum ada tujuan tabungan</div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {savings.map(s => {
          const pct = Math.min(100, (s.current_balance / s.target_amount) * 100) || 0
          return (
            <div key={s.id} className={`card ${pct >= 100 ? 'border-green-200' : ''}`}>
              <div className="flex items-start justify-between mb-2">
                <h3 className="font-semibold text-gray-800">{s.name}</h3>
                {pct >= 100 && <span className="badge bg-green-100 text-green-700">🎉 Tercapai!</span>}
              </div>

              <div className="mb-2">
                <div className="flex justify-between text-xs text-gray-500 mb-1">
                  <span>{rupiah(s.current_balance)}</span>
                  <span>{rupiah(s.target_amount)}</span>
                </div>
                <div className="h-3 bg-gray-100 rounded-full overflow-hidden">
                  <div className="h-full bg-green-500 rounded-full transition-all" style={{ width: `${pct}%` }} />
                </div>
                <div className="text-xs text-gray-400 mt-0.5 text-right">{pct.toFixed(0)}%</div>
              </div>

              <div className="grid grid-cols-2 gap-2 text-xs text-gray-500 mt-3">
                <div>
                  <span className="text-gray-400">Target: </span>
                  <span>{fmtDate(s.target_date)}</span>
                </div>
                <div>
                  <span className="text-gray-400">Perlu: </span>
                  <span className="font-medium text-blue-600">{rupiah(s.monthly_required)}/bln</span>
                </div>
              </div>

              <div className="flex gap-2 mt-3 pt-3 border-t border-gray-100">
                <button
                  onClick={() => { setDepositSaving(s); setDepForm({ amount: '', date: today(), note: '' }) }}
                  className="btn-primary flex-1 text-xs py-1.5"
                >
                  + Setor
                </button>
                <button onClick={() => openEdit(s)} className="btn-outline text-xs py-1.5 px-3">✏️</button>
                <button onClick={() => handleDelete(s.id)} className="btn-outline text-xs py-1.5 px-3 text-red-500">🗑</button>
              </div>
            </div>
          )
        })}
      </div>

      {/* Modal Tambah/Edit */}
      {showCreate && (
        <Modal title={editSaving ? 'Edit Tujuan Tabungan' : 'Tujuan Tabungan Baru'} onClose={() => setShowCreate(false)}>
          <form onSubmit={handleSubmit} className="space-y-3">
            <div>
              <label className="label">Nama Tujuan</label>
              <input className="input" placeholder="cth: Dana Pendidikan, Liburan" value={form.name}
                onChange={e => setForm(f => ({ ...f, name: e.target.value }))} required />
            </div>
            <div>
              <label className="label">Target (Rp)</label>
              <input className="input" type="number" min="1" step="any" value={form.target_amount}
                onChange={e => setForm(f => ({ ...f, target_amount: e.target.value }))} required />
            </div>
            {!editSaving && (
              <div>
                <label className="label">Saldo Awal (Rp)</label>
                <input className="input" type="number" min="0" step="any" value={form.initial_balance}
                  onChange={e => setForm(f => ({ ...f, initial_balance: e.target.value }))} />
              </div>
            )}
            <div>
              <label className="label">Target Tanggal</label>
              <input className="input" type="date" value={form.target_date}
                onChange={e => setForm(f => ({ ...f, target_date: e.target.value }))} required />
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

      {/* Modal Setoran */}
      {depositSaving && (
        <Modal title={`Setor ke — ${depositSaving.name}`} onClose={() => setDepositSaving(null)}>
          <form onSubmit={handleDeposit} className="space-y-3">
            <div>
              <label className="label">Jumlah Setoran (Rp)</label>
              <input className="input" type="number" min="1" step="any" value={depForm.amount}
                onChange={e => setDepForm(f => ({ ...f, amount: e.target.value }))} required />
            </div>
            <div>
              <label className="label">Tanggal</label>
              <input className="input" type="date" value={depForm.date}
                onChange={e => setDepForm(f => ({ ...f, date: e.target.value }))} required />
            </div>
            <div>
              <label className="label">Catatan</label>
              <input className="input" placeholder="Opsional" value={depForm.note}
                onChange={e => setDepForm(f => ({ ...f, note: e.target.value }))} />
            </div>
            <div className="p-2.5 bg-green-50 border border-green-100 rounded-lg text-xs text-green-700">
              🎯 Setoran ini otomatis dicatat sebagai pengeluaran <strong>"{depositSaving.name}"</strong> di halaman Transaksi atas nama <strong>{username}</strong>.
            </div>
            <div className="flex gap-2 pt-1">
              <button type="submit" disabled={loading} className="btn-primary flex-1">
                {loading ? 'Menyimpan...' : 'Catat Setoran'}
              </button>
              <button type="button" onClick={() => setDepositSaving(null)} className="btn-outline flex-1">Batal</button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  )
}
