import { api, session } from '../api.js'
import { Brand, Icon, PANEL_NAME } from '../ui.jsx'
import { navigate } from '../App.jsx'

export default function Layout({ user, title, subtitle, action, onLogout, onSync, syncing, children }) {
  async function logout() {
    try { await api('/api/auth/logout', { method: 'POST' }) } catch {}
    session.clear()
    onLogout?.()
    window.dispatchEvent(new Event('session-expired'))
  }
  return <div class="app-shell">
    <header class="topbar"><Brand compact /><div class="topbar-user">{onSync && <button class={`icon-button sync-button${syncing ? ' button-spinning' : ''}`} onClick={onSync} disabled={syncing} title="Перезагрузить AWG конфиг" aria-label="Перезагрузить AWG конфиг"><Icon name="sync" /></button>}<span class="user-dot">{user.username[0].toUpperCase()}</span><div><strong>{user.username}</strong><small>{user.role === 'admin' ? 'Администратор' : 'Пользователь'}</small></div><button class="icon-button" onClick={logout} aria-label="Выйти"><Icon name="logout" /></button></div></header>
    <main class="page"><div class="page-heading"><div><span class="eyebrow">{PANEL_NAME}</span><h1>{title}</h1><p>{subtitle}</p></div>{action}</div>{children}</main>
  </div>
}
