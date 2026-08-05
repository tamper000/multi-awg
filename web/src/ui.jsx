import { useEffect, useState } from 'preact/hooks'

export function Icon({ name, size = 20 }) {
  const paths = {
    bolt: <path d="M13 2 4 14h7l-1 8 9-12h-7l1-8Z" />,
    plus: <path d="M12 5v14M5 12h14" />,
    arrow: <path d="m9 18 6-6-6-6" />,
    copy: <><rect x="9" y="9" width="11" height="11" rx="2" /><path d="M15 9V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v7a2 2 0 0 0 2 2h3" /></>,
    download: <><path d="M12 3v12m0 0 5-5m-5 5-5-5" /><path d="M5 21h14" /></>,
    trash: <><path d="M4 7h16M9 7V4h6v3m3 0-1 14H7L6 7" /><path d="M10 11v6m4-6v6" /></>,
    logout: <><path d="M10 17l5-5-5-5m5 5H3" /><path d="M14 3h7v18h-7" /></>,
    users: <><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" /><path d="M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" /></>,
    key: <><circle cx="8" cy="15" r="4" /><path d="m11 12 9-9m-4 4 3 3" /></>,
    link: <><path d="M10 13a5 5 0 0 0 7.07.07l2-2a5 5 0 0 0-7.07-7.07l-1.15 1.15" /><path d="M14 11a5 5 0 0 0-7.07-.07l-2 2A5 5 0 0 0 12 20l1.15-1.15" /></>,
    back: <path d="m15 18-6-6 6-6" />,
    eye: <><path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z" /><circle cx="12" cy="12" r="3" /></>,
    sync: <path d="M20 11a8 8 0 0 0-14.9-3M4 4v4h4M4 13a8 8 0 0 0 14.9 3M20 20v-4h-4" />,
  }
  return <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">{paths[name]}</svg>
}

export const PANEL_NAME = import.meta.env.VITE_PANEL_NAME || 'Банановый Ускоритель'

export function Brand({ compact = false }) {
  return <div class={`brand ${compact ? 'brand-compact' : ''}`}><span class="brand-mark"><Icon name="bolt" /></span><span>{PANEL_NAME}</span></div>
}

export function Loader() {
  return <div class="loader-wrap"><span class="loader" /><span>Загружаем данные</span></div>
}

export function Notice({ children, kind = 'error' }) {
  return children ? <div class={`notice notice-${kind}`} role="alert">{children}</div> : null
}

export function Toast() {
  const [message, setMessage] = useState('')
  useEffect(() => {
    let timer
    const show = (event) => {
      setMessage(event.detail)
      clearTimeout(timer)
      timer = setTimeout(() => setMessage(''), 3500)
    }
    addEventListener('app-error', show)
    return () => { removeEventListener('app-error', show); clearTimeout(timer) }
  }, [])
  return message ? <div class="toast"><Notice>{message}</Notice></div> : null
}

export function Modal({ title, children, onClose }) {
  return <div class="modal-backdrop" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
    <section class="modal" role="dialog" aria-modal="true" aria-labelledby="modal-title">
      <button class="modal-close" type="button" onClick={onClose} aria-label="Закрыть">×</button>
      <h2 id="modal-title">{title}</h2>
      {children}
    </section>
  </div>
}

export function Empty({ title, text }) {
  return <div class="empty"><span><Icon name="bolt" size={28} /></span><h3>{title}</h3><p>{text}</p></div>
}

export function formatBytes(value) {
  if (value === undefined || value === null) return '—'
  if (value === 0) return '0 Б'
  const units = ['Б', 'КБ', 'МБ', 'ГБ', 'ТБ']
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  return `${(value / 1024 ** index).toFixed(index ? 1 : 0)} ${units[index]}`
}

export function formatDate(value, withTime = false) {
  if (!value) return 'Без срока'
  return new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium', ...(withTime && { timeStyle: 'short' }) }).format(new Date(value))
}

export function copyText(value, setCopied) {
  navigator.clipboard.writeText(value).then(() => {
    setCopied(true)
    setTimeout(() => setCopied(false), 1600)
  })
}
