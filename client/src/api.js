import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  withCredentials: true,
  headers: { 'Content-Type': 'application/json' },
})

// ── Wallet API ────────────────────────────────────────────────
export const walletAPI = {
  list:     ()       => api.get('/wallets'),
  create:   (data)   => api.post('/wallets', data),
  update:   (id, d)  => api.put(`/wallets/${id}`, d),
  delete:   (id)     => api.delete(`/wallets/${id}`),
  transfer: (data)   => api.post('/wallets/transfer', data),
}

// Intercept 401 → set flag dan biarkan React Router yang handle redirect,
// BUKAN window.location.href (yang menyebabkan full-page refresh loop).
// Semua endpoint auth dibiarkan lolos agar catch() di komponen bisa handle.
let _navigateToLogin = null

export function setNavigateToLogin(fn) {
  _navigateToLogin = fn
}

api.interceptors.response.use(
  (res) => res,
  (err) => {
    const url = err.config?.url || ''
    const isAuthEndpoint = url.startsWith('/auth/')
    if (err.response?.status === 401 && !isAuthEndpoint) {
      if (_navigateToLogin) {
        _navigateToLogin()
      } else {
        // Fallback jika React Router belum siap
        window.location.href = '/login'
      }
    }
    return Promise.reject(err)
  }
)

export default api
