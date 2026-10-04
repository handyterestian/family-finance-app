import { useState } from 'react'
import api from '../api.js'
import Modal from '../components/Modal.jsx'
import { usePageFocus } from '../hooks/usePageFocus.js'

export default function CategoriesPage() {
  const [cats, setCats] = useState([])
  const [showForm, setShowForm] = useState(false)
  const [editCat, setEditCat] = useState(null)
  const [form, setForm] = useState({ name: '', type: 'both' })
  const [loading, setLoading] = useState(false)

  const load = () => api.get('/categories').then(r => setCats(r.data.categories || []))
  usePageFocus(load)

  const openCreate = () => { setForm({ name: '', type: 'both' }); setEditCat(null); setShowForm(true) }
  const openEdit = (c) => { setForm({ name: c.name, type: c.type, new_name: c.name }); setEditCat(c); setShowForm(true) }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setLoading(true)
    try {
      if (editCat) {
        await api.put(`/categories/${editCat.name}`, { new_name: form.name, type: form.type })
      } else {
        await api.post('/categories', { name: form.name, type: form.type })
      }
      setShowForm(false)
      load()
    } catch (err) {
      alert(err.response?.data?.error || 'Gagal menyimpan kategori')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (name) => {
    if (!confirm(`Hapus kategori "${name}"?`)) return
    try {
      await api.delete(`/categories/${encodeURIComponent(name)}`)
      load()
    } catch (err) {
      alert(err.response?.data?.error || 'Gagal menghapus kategori')
    }
  }

  const typeBadge = (t) => {
    if (t === 'income')  return <span className="badge bg-green-100 text-green-700">Pemasukan</span>
    if (t === 'expense') return <span className="badge bg-red-100 text-red-600">Pengeluaran</span>
    return <span className="badge bg-gray-100 text-gray-600">Keduanya</span>
  }

  return (
    <div className="p-4 sm:p-6 space-y-5">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-gray-800">🏷️ Kategori</h1>
        <button onClick={openCreate} className="btn-primary">+ Tambah Kategori</button>
      </div>

      <div className="card overflow-x-auto p-0">
        <table className="w-full text-sm min-w-[320px]">
          <thead>
            <tr className="bg-gray-50 border-b border-gray-100">
              <th className="text-left px-4 py-3 font-medium text-gray-600">Nama Kategori</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">Tipe</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody>
            {cats.length === 0 && (
              <tr><td colSpan={3} className="text-center py-8 text-gray-400">Belum ada kategori</td></tr>
            )}
            {cats.map(c => (
              <tr key={c.id} className="border-b border-gray-50 hover:bg-gray-50">
                <td className="px-4 py-2.5 font-medium">{c.name}</td>
                <td className="px-4 py-2.5">{typeBadge(c.type)}</td>
                <td className="px-4 py-2.5">
                  <div className="flex gap-1 justify-end">
                    <button onClick={() => openEdit(c)} className="btn-ghost text-xs px-2 py-1">✏️</button>
                    <button onClick={() => handleDelete(c.name)} className="btn-ghost text-xs px-2 py-1 text-red-500">🗑</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showForm && (
        <Modal title={editCat ? 'Edit Kategori' : 'Tambah Kategori'} onClose={() => setShowForm(false)}>
          <form onSubmit={handleSubmit} className="space-y-3">
            <div>
              <label className="label">Nama Kategori</label>
              <input className="input" placeholder="cth: Olahraga, Asuransi" value={form.name}
                onChange={e => setForm(f => ({ ...f, name: e.target.value }))} required />
            </div>
            <div>
              <label className="label">Tipe</label>
              <select className="input" value={form.type} onChange={e => setForm(f => ({ ...f, type: e.target.value }))}>
                <option value="income">Pemasukan</option>
                <option value="expense">Pengeluaran</option>
                <option value="both">Keduanya</option>
              </select>
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
    </div>
  )
}
