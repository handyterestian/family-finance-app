import { useState } from 'react'
import api from '../api.js'

export default function AdminPage() {
  const [backupLoading, setBackupLoading] = useState(false)
  const [restoreLoading, setRestoreLoading] = useState(false)
  const [restoreMsg, setRestoreMsg] = useState(null) // { type: 'ok'|'err', text }

  const handleBackup = async () => {
    setBackupLoading(true)
    try {
      const resp = await api.get('/backup', { responseType: 'blob' })
      const url = URL.createObjectURL(resp.data)
      const a = document.createElement('a')
      a.href = url
      const today = new Date().toISOString().slice(0, 10)
      a.download = `backup-${today}.json`
      a.click()
      URL.revokeObjectURL(url)
    } catch (e) {
      alert('Gagal membuat backup: ' + (e.response?.data?.error || e.message))
    } finally {
      setBackupLoading(false)
    }
  }

  const handleRestore = async (e) => {
    const file = e.target.files?.[0]
    if (!file) return
    if (!confirm(`Restore dari file "${file.name}"?\nSemua data saat ini akan digantikan dengan isi file backup!`)) {
      e.target.value = ''
      return
    }
    setRestoreLoading(true)
    setRestoreMsg(null)
    try {
      const form = new FormData()
      form.append('file', file)
      const resp = await api.post('/restore', form, { headers: { 'Content-Type': 'multipart/form-data' } })
      setRestoreMsg({ type: 'ok', text: resp.data?.message || 'Restore berhasil!' })
    } catch (e) {
      setRestoreMsg({ type: 'err', text: 'Gagal restore: ' + (e.response?.data?.error || e.message) })
    } finally {
      setRestoreLoading(false)
      e.target.value = ''
    }
  }

  return (
    <div className="p-6 max-w-xl space-y-6">
      <h1 className="text-xl font-bold text-gray-800">⚙️ Admin</h1>

      {/* Backup */}
      <div className="card space-y-3">
        <div>
          <h2 className="text-sm font-semibold text-gray-700">💾 Backup Data & Auto Backup</h2>
          <p className="text-xs text-gray-400 mt-0.5">
            Unduh seluruh data keluarga sebagai file JSON. Simpan file ini sebagai cadangan manual.
          </p>
          <div className="mt-2.5 p-2.5 bg-blue-50 border border-blue-100 rounded-lg text-xs text-blue-700 space-y-1">
            <p className="font-semibold flex items-center gap-1">🔄 Auto Backup Mingguan Aktif</p>
            <p className="text-gray-600">
              Sistem secara otomatis membuat cadangan data inkremental/lengkap di server setiap 7 hari sekali. File di server ditimpa (replace) tiap minggunya guna menghemat kapasitas penyimpanan (storage).
            </p>
          </div>
        </div>
        <button
          onClick={handleBackup}
          disabled={backupLoading}
          className="btn-primary text-sm"
        >
          {backupLoading ? 'Mengunduh...' : '💾 Unduh Backup Sekarang'}
        </button>
      </div>

      {/* Restore */}
      <div className="card space-y-3">
        <div>
          <h2 className="text-sm font-semibold text-gray-700">📂 Restore Data</h2>
          <p className="text-xs text-gray-400 mt-0.5">
            Pulihkan data dari file backup JSON. <span className="text-red-500 font-medium">Semua data saat ini akan digantikan.</span>
          </p>
        </div>
        {restoreMsg && (
          <div className={`text-xs px-3 py-2 rounded-lg ${restoreMsg.type === 'ok' ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-600'}`}>
            {restoreMsg.text}
          </div>
        )}
        <label className={`btn-outline text-sm cursor-pointer inline-block ${restoreLoading ? 'opacity-60 pointer-events-none' : ''}`}>
          {restoreLoading ? '⏳ Memulihkan...' : '📂 Pilih File Backup (.json)'}
          <input type="file" accept=".json" className="hidden" onChange={handleRestore} disabled={restoreLoading} />
        </label>
      </div>
    </div>
  )
}
