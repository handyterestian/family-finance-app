import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import api from '../api.js'

export default function RegisterPage() {
  const [form, setForm] = useState({ username: '', email: '', password: '', family_name: '' })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const nav = useNavigate()

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await api.post('/auth/register', form)
      nav('/login')
    } catch (err) {
      setError(err.response?.data?.error || 'Registrasi gagal. Coba lagi.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100">
      <div className="w-full max-w-md">
        <div className="card shadow-lg">
          <div className="text-center mb-6">
            <div className="text-4xl mb-2">🏠</div>
            <h2 className="text-2xl font-bold text-gray-800">Buat Akun Keluarga</h2>
            <p className="text-sm text-gray-500 mt-1">Semua anggota berbagi data yang sama</p>
          </div>

          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-600">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="label">Nama Keluarga</label>
              <input
                className="input"
                type="text"
                placeholder="cth: Keluarga Budi"
                value={form.family_name}
                onChange={e => setForm(f => ({ ...f, family_name: e.target.value }))}
                required
              />
              <p className="text-xs text-gray-400 mt-1">Anggota keluarga lain bisa bergabung dengan akun yang sama</p>
            </div>
            <div>
              <label className="label">Username</label>
              <input
                className="input"
                type="text"
                placeholder="cth: Ayah"
                value={form.username}
                onChange={e => setForm(f => ({ ...f, username: e.target.value }))}
                required
              />
            </div>
            <div>
              <label className="label">Email</label>
              <input
                className="input"
                type="email"
                placeholder="email@contoh.com"
                value={form.email}
                onChange={e => setForm(f => ({ ...f, email: e.target.value }))}
                required
              />
            </div>
            <div>
              <label className="label">Password</label>
              <input
                className="input"
                type="password"
                placeholder="Minimal 8 karakter"
                value={form.password}
                onChange={e => setForm(f => ({ ...f, password: e.target.value }))}
                required
                minLength={8}
              />
            </div>
            <button
              type="submit"
              disabled={loading}
              className="btn-primary w-full py-2.5"
            >
              {loading ? 'Mendaftar...' : 'Buat Akun'}
            </button>
          </form>

          <p className="text-center text-sm text-gray-500 mt-5">
            Sudah punya akun?{' '}
            <Link to="/login" className="text-blue-600 hover:underline font-medium">
              Masuk di sini
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
