import { useState, useEffect } from 'react'
import { walletAPI } from '../api.js'
import { rupiah as fmt, today } from '../utils.js'
import { usePageFocus } from '../hooks/usePageFocus.js'
import { useCurrentUser } from '../App.jsx'

const WALLET_TYPES = [
  { value: 'cash',       label: '💵 Tunai' },
  { value: 'bank',       label: '🏦 Bank' },
  { value: 'e-wallet',   label: '📱 E-Wallet' },
  { value: 'investment', label: '📈 Investasi' },
  { value: 'other',      label: '🗂 Lainnya' },
]

const COLORS = [
  '#6366f1','#3b82f6','#10b981','#f59e0b','#ef4444',
  '#8b5cf6','#ec4899','#14b8a6','#f97316','#64748b',
]

function typeLabel(type) {
  return WALLET_TYPES.find(t => t.value === type)?.label || type
}

function WalletCard({ w, onEdit, onDelete, onTransfer, onAdjust }) {
  return (
    <div className="bg-white rounded-xl border border-gray-200 p-4 flex flex-col gap-3">
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full flex-shrink-0" style={{ background: w.color || '#6366f1' }} />
          <span className="font-semibold text-gray-800">{w.name}</span>
        </div>
        <span className="text-xs text-gray-400 bg-gray-100 px-2 py-0.5 rounded-full">{typeLabel(w.type)}</span>
      </div>
      <div className="text-2xl font-bold" style={{ color: w.color || '#6366f1' }}>
        {fmt(w.balance)}
      </div>
      {w.note && <p className="text-xs text-gray-500">{w.note}</p>}
      <div className="flex gap-2 pt-1 border-t border-gray-100">
        <button onClick={() => onTransfer(w)}
          className="flex-1 text-xs py-1.5 rounded-lg bg-indigo-50 text-indigo-700 hover:bg-indigo-100 font-medium">
          ⇄ Transfer
        </button>
        <button onClick={() => onAdjust(w)}
          className="flex-1 text-xs py-1.5 rounded-lg bg-amber-50 text-amber-700 hover:bg-amber-100 font-medium">
          ⚖️ Sesuaikan
        </button>
        <button onClick={() => onEdit(w)}
          className="text-xs py-1.5 px-3 rounded-lg bg-gray-100 text-gray-700 hover:bg-gray-200">
          ✏️
        </button>
        <button onClick={() => onDelete(w)}
          className="text-xs py-1.5 px-3 rounded-lg bg-red-50 text-red-600 hover:bg-red-100">
          🗑
        </button>
      </div>
    </div>
  )
}

function WalletModal({ initial, onSave, onClose }) {
  const [form, setForm] = useState({
    name: initial?.name || '',
    type: initial?.type || 'cash',
    balance: initial?.balance ?? 0,
    color: initial?.color || '#6366f1',
    note: initial?.note || '',
  })
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState('')

  const set = (k, v) => setForm(p => ({ ...p, [k]: v }))

  async function handleSubmit(e) {
    e.preventDefault()
    setSaving(true); setErr('')
    try {
      await onSave(form)
      onClose()
    } catch (e) {
      setErr(e.response?.data?.error || 'Gagal menyimpan dompet')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl shadow-xl w-full max-w-md">
        <div className="flex justify-between items-center px-5 pt-5 pb-3 border-b">
          <h3 className="font-semibold text-gray-800">{initial ? 'Edit Dompet' : 'Tambah Dompet'}</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-xl">×</button>
        </div>
        <form onSubmit={handleSubmit} className="p-5 flex flex-col gap-4">
          {err && <div className="text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">{err}</div>}

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Nama Dompet</label>
            <input value={form.name} onChange={e => set('name', e.target.value)} required
              className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-300 outline-none" />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Tipe</label>
              <select value={form.type} onChange={e => set('type', e.target.value)}
                className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-300 outline-none">
                {WALLET_TYPES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                {initial ? 'Warna' : 'Saldo Awal'}
              </label>
              {initial ? (
                <div className="flex gap-1 flex-wrap">
                  {COLORS.map(c => (
                    <button type="button" key={c}
                      onClick={() => set('color', c)}
                      className={`w-6 h-6 rounded-full border-2 ${form.color === c ? 'border-gray-800 scale-110' : 'border-transparent'}`}
                      style={{ background: c }} />
                  ))}
                </div>
              ) : (
                <input type="number" value={form.balance} onChange={e => set('balance', +e.target.value)} min="0"
                  className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-300 outline-none" />
              )}
            </div>
          </div>

          {!initial && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Warna</label>
              <div className="flex gap-1 flex-wrap">
                {COLORS.map(c => (
                  <button type="button" key={c}
                    onClick={() => set('color', c)}
                    className={`w-6 h-6 rounded-full border-2 ${form.color === c ? 'border-gray-800 scale-110' : 'border-transparent'}`}
                    style={{ background: c }} />
                ))}
              </div>
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Catatan (opsional)</label>
            <input value={form.note} onChange={e => set('note', e.target.value)}
              className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-300 outline-none" />
          </div>

          <div className="flex gap-3 pt-1">
            <button type="button" onClick={onClose}
              className="flex-1 border border-gray-300 text-gray-700 rounded-xl py-2 text-sm hover:bg-gray-50">
              Batal
            </button>
            <button type="submit" disabled={saving}
              className="flex-1 bg-indigo-600 text-white rounded-xl py-2 text-sm font-medium hover:bg-indigo-700 disabled:opacity-60">
              {saving ? 'Menyimpan…' : 'Simpan'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

function AdjustModal({ wallet, username, onSave, onClose }) {
  const [form, setForm] = useState({
    new_balance: wallet.balance ?? 0,
    note: '',
    date: today(),
  })
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState('')

  const set = (k, v) => setForm(p => ({ ...p, [k]: v }))

  const delta = Number(form.new_balance) - wallet.balance
  const deltaAbs = Math.abs(delta)

  async function handleSubmit(e) {
    e.preventDefault()
    setSaving(true); setErr('')
    try {
      await onSave(wallet.id, { ...form, new_balance: Number(form.new_balance), member: username })
      onClose()
    } catch (e) {
      setErr(e.response?.data?.error || 'Gagal menyesuaikan saldo')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl shadow-xl w-full max-w-md">
        <div className="flex justify-between items-center px-5 pt-5 pb-3 border-b">
          <div>
            <h3 className="font-semibold text-gray-800">⚖️ Sesuaikan Saldo</h3>
            <p className="text-xs text-gray-400 mt-0.5">{wallet.name}</p>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-xl">×</button>
        </div>
        <form onSubmit={handleSubmit} className="p-5 flex flex-col gap-4">
          {err && <div className="text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">{err}</div>}

          {/* Info saldo sekarang */}
          <div className="bg-gray-50 rounded-xl px-4 py-3 flex items-center justify-between text-sm">
            <span className="text-gray-500">Saldo saat ini</span>
            <span className="font-semibold text-gray-800">{fmt(wallet.balance)}</span>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Saldo Baru (Rp)</label>
            <input
              type="number" min="0" step="any" required
              value={form.new_balance}
              onChange={e => set('new_balance', e.target.value)}
              className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-amber-300 outline-none"
            />
          </div>

          {/* Preview selisih */}
          {delta !== 0 && (
            <div className={`rounded-xl px-4 py-3 text-sm flex items-center justify-between ${
              delta > 0 ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'
            }`}>
              <span>{delta > 0 ? '↑ Penambahan saldo' : '↓ Pengurangan saldo'}</span>
              <span className="font-semibold">{fmt(deltaAbs)}</span>
            </div>
          )}
          {delta === 0 && form.new_balance !== '' && (
            <div className="bg-gray-50 rounded-xl px-4 py-3 text-sm text-gray-500 text-center">
              Saldo tidak berubah
            </div>
          )}

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Tanggal</label>
              <input type="date" required
                value={form.date}
                onChange={e => set('date', e.target.value)}
                className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-amber-300 outline-none"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Catatan (opsional)</label>
              <input
                value={form.note}
                onChange={e => set('note', e.target.value)}
                placeholder="Cth: koreksi saldo"
                className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-amber-300 outline-none"
              />
            </div>
          </div>

          <p className="text-xs text-gray-400">
            Penyesuaian akan otomatis tercatat sebagai transaksi &quot;Penyesuaian Saldo&quot; di halaman Transaksi.
          </p>

          <div className="flex gap-3 pt-1">
            <button type="button" onClick={onClose}
              className="flex-1 border border-gray-300 text-gray-700 rounded-xl py-2 text-sm hover:bg-gray-50">
              Batal
            </button>
            <button type="submit" disabled={saving || delta === 0}
              className="flex-1 bg-amber-500 text-white rounded-xl py-2 text-sm font-medium hover:bg-amber-600 disabled:opacity-50">
              {saving ? 'Memproses…' : 'Simpan Penyesuaian'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

function TransferModal({ wallets, initial, onSave, onClose }) {
  const today = new Date().toISOString().slice(0, 10)
  const [form, setForm] = useState({
    from_wallet_id: initial?.id || '',
    to_wallet_id: '',
    amount: '',
    note: '',
    date: today,
  })
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState('')

  const set = (k, v) => setForm(p => ({ ...p, [k]: v }))

  async function handleSubmit(e) {
    e.preventDefault()
    setSaving(true); setErr('')
    try {
      await onSave({ ...form, amount: +form.amount })
      onClose()
    } catch (e) {
      setErr(e.response?.data?.error || 'Gagal transfer')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl shadow-xl w-full max-w-md">
        <div className="flex justify-between items-center px-5 pt-5 pb-3 border-b">
          <h3 className="font-semibold text-gray-800">⇄ Transfer Antar Dompet</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-xl">×</button>
        </div>
        <form onSubmit={handleSubmit} className="p-5 flex flex-col gap-4">
          {err && <div className="text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">{err}</div>}

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Dari Dompet</label>
              <select value={form.from_wallet_id} onChange={e => set('from_wallet_id', e.target.value)} required
                className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-300 outline-none">
                <option value="">Pilih…</option>
                {wallets.map(w => (
                  <option key={w.id} value={w.id}>{w.name} ({fmt(w.balance)})</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Ke Dompet</label>
              <select value={form.to_wallet_id} onChange={e => set('to_wallet_id', e.target.value)} required
                className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-300 outline-none">
                <option value="">Pilih…</option>
                {wallets.filter(w => w.id !== form.from_wallet_id).map(w => (
                  <option key={w.id} value={w.id}>{w.name} ({fmt(w.balance)})</option>
                ))}
              </select>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Jumlah</label>
              <input type="number" value={form.amount} onChange={e => set('amount', e.target.value)}
                required min="1"
                className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-300 outline-none" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Tanggal</label>
              <input type="date" value={form.date} onChange={e => set('date', e.target.value)} required
                className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-300 outline-none" />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Catatan</label>
            <input value={form.note} onChange={e => set('note', e.target.value)}
              className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-300 outline-none" />
          </div>

          <div className="flex gap-3 pt-1">
            <button type="button" onClick={onClose}
              className="flex-1 border border-gray-300 text-gray-700 rounded-xl py-2 text-sm hover:bg-gray-50">
              Batal
            </button>
            <button type="submit" disabled={saving}
              className="flex-1 bg-indigo-600 text-white rounded-xl py-2 text-sm font-medium hover:bg-indigo-700 disabled:opacity-60">
              {saving ? 'Memproses…' : 'Transfer'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default function WalletPage() {
  const currentUser = useCurrentUser()
  const username = currentUser?.username || ''

  const [wallets, setWallets] = useState([])
  const [loading, setLoading] = useState(true)
  const [modal, setModal] = useState(null) // null | 'create' | {wallet} | 'transfer' | {adjust,wallet}
  const [transferTarget, setTransferTarget] = useState(null)
  const [adjustTarget, setAdjustTarget] = useState(null)
  const [deleteConfirm, setDeleteConfirm] = useState(null)

  async function load() {
    try {
      const res = await walletAPI.list()
      setWallets(res.data.wallets || [])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])
  usePageFocus(load)

  const totalBalance = wallets.reduce((s, w) => s + (Number(w.balance) || 0), 0)

  async function handleCreate(form) {
    await walletAPI.create(form)
    await load()
  }

  async function handleUpdate(form) {
    await walletAPI.update(modal.id, form)
    await load()
  }

  async function handleDelete(w) {
    if (!deleteConfirm || deleteConfirm.id !== w.id) {
      setDeleteConfirm(w); return
    }
    await walletAPI.delete(w.id)
    setDeleteConfirm(null)
    await load()
  }

  async function handleTransfer(form) {
    await walletAPI.transfer(form)
    await load()
  }

  async function handleAdjust(id, form) {
    await walletAPI.adjust(id, form)
    await load()
  }

  if (loading) return (
    <div className="flex items-center justify-center h-64">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600" />
    </div>
  )

  return (
    <div className="max-w-4xl mx-auto p-4 sm:p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-gray-800">💰 Dompet</h1>
          <p className="text-sm text-gray-500">Kelola sumber dana keluarga</p>
        </div>
        <button onClick={() => setModal('create')}
          className="bg-indigo-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-indigo-700">
          + Tambah Dompet
        </button>
      </div>

      {/* Total saldo */}
      <div className="bg-gradient-to-r from-indigo-600 to-indigo-400 rounded-2xl p-5 text-white">
        <p className="text-sm opacity-80">Total Saldo Semua Dompet</p>
        <p className="text-3xl font-bold mt-1">{fmt(totalBalance)}</p>
        <p className="text-sm opacity-70 mt-1">{wallets.length} dompet aktif</p>
      </div>

      {/* Daftar dompet */}
      {wallets.length === 0 ? (
        <div className="text-center py-16 text-gray-400">
          <p className="text-4xl mb-3">👛</p>
          <p className="font-medium">Belum ada dompet</p>
          <p className="text-sm">Tambahkan dompet untuk melacak saldo per sumber dana</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {wallets.map(w => (
            <WalletCard key={w.id} w={w}
              onEdit={w => setModal(w)}
              onDelete={handleDelete}
              onTransfer={w => { setTransferTarget(w); setModal('transfer') }}
              onAdjust={w => { setAdjustTarget(w); setModal('adjust') }}
            />
          ))}
        </div>
      )}

      {/* Konfirmasi hapus */}
      {deleteConfirm && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl shadow-xl p-6 max-w-sm w-full text-center">
            <p className="text-lg font-semibold text-gray-800 mb-2">Hapus Dompet?</p>
            <p className="text-sm text-gray-500 mb-5">
              Dompet <strong>{deleteConfirm.name}</strong> akan dihapus. Transaksi yang terkait tidak akan terhapus.
            </p>
            <div className="flex gap-3">
              <button onClick={() => setDeleteConfirm(null)}
                className="flex-1 border border-gray-300 text-gray-700 rounded-xl py-2 text-sm hover:bg-gray-50">
                Batal
              </button>
              <button onClick={() => handleDelete(deleteConfirm)}
                className="flex-1 bg-red-600 text-white rounded-xl py-2 text-sm font-medium hover:bg-red-700">
                Hapus
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Modal tambah/edit */}
      {(modal === 'create' || (modal && typeof modal === 'object' && !modal._transfer)) && modal !== 'transfer' && (
        <WalletModal
          initial={modal === 'create' ? null : modal}
          onSave={modal === 'create' ? handleCreate : handleUpdate}
          onClose={() => setModal(null)}
        />
      )}

      {/* Modal transfer */}
      {modal === 'transfer' && (
        <TransferModal
          wallets={wallets}
          initial={transferTarget}
          onSave={handleTransfer}
          onClose={() => { setModal(null); setTransferTarget(null) }}
        />
      )}

      {/* Modal penyesuaian saldo */}
      {modal === 'adjust' && adjustTarget && (
        <AdjustModal
          wallet={adjustTarget}
          username={username}
          onSave={handleAdjust}
          onClose={() => { setModal(null); setAdjustTarget(null) }}
        />
      )}
    </div>
  )
}
