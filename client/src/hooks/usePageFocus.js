import { useEffect, useRef } from 'react'
import { useLocation } from 'react-router-dom'

/**
 * Memanggil `callback` setiap kali halaman ini menjadi aktif:
 *  - saat pertama kali di-mount
 *  - setiap kali pengguna navigasi kembali ke halaman ini (pathname cocok)
 *
 * Cara pakai:
 *   usePageFocus(load)          — panggil ulang saat halaman difokuskan
 *   usePageFocus(load, [dep])   — seperti useEffect biasa, plus re-run saat navigasi masuk
 */
export function usePageFocus(callback, deps = []) {
  const location = useLocation()
  const callbackRef = useRef(callback)

  // Selalu update ref ke versi terbaru callback agar tidak perlu callback masuk ke deps
  useEffect(() => {
    callbackRef.current = callback
  })

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    callbackRef.current()
  // location.pathname sebagai dep: setiap kali URL berubah ke halaman ini, re-run
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.pathname, ...deps])
}
