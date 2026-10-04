// Format angka ke Rupiah Indonesia
export const rupiah = (n) =>
  new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n || 0)

// Format tanggal ISO ke dd/mm/yyyy
export const fmtDate = (d) => {
  if (!d) return '-'
  const [y, m, day] = d.substring(0, 10).split('-')
  return `${day}/${m}/${y}`
}

// Bulan saat ini dalam format YYYY-MM
export const thisMonth = () => new Date().toISOString().substring(0, 7)

// Tanggal hari ini dalam format YYYY-MM-DD
export const today = () => new Date().toISOString().substring(0, 10)

// Progress bar percentage capped 0-100
export const pct = (val, max) => (max > 0 ? Math.min(100, (val / max) * 100) : 0)

// Warna progress bar berdasarkan persentase
export const progressColor = (p) => {
  if (p >= 100) return 'bg-red-500'
  if (p >= 80)  return 'bg-yellow-400'
  return 'bg-green-500'
}

// Warna skor kesehatan
export const scoreColor = (score, max) => {
  const r = score / max
  if (r >= 0.75) return 'text-green-600'
  if (r >= 0.5)  return 'text-yellow-500'
  return 'text-red-500'
}
