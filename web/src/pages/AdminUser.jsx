import { useEffect, useState } from 'preact/hooks'
import { api, jsonBody } from '../api.js'
import { formatBytes, formatDate, Icon, Loader, Modal, Notice } from '../ui.jsx'
import { navigate } from '../App.jsx'
import Layout from './Layout.jsx'

export default function AdminUser({ username, user }) {
  const [data, setData] = useState(null)
  const [error, setError] = useState('')
  const [extendOpen, setExtendOpen] = useState(false)
  async function load() {
    try { setData(await api(`/api/admin/users/${encodeURIComponent(username)}`)); setError('') } catch (err) { setError(err.message) }
  }
  useEffect(() => { load() }, [username])

  async function extend(event) {
    event.preventDefault()
    const days = Number(new FormData(event.currentTarget).get('days'))
    try { await api(`/api/admin/users/${encodeURIComponent(username)}`, { method: 'PATCH', body: jsonBody({ days }) }); setExtendOpen(false); load() } catch (err) { setError(err.message) }
  }
  async function remove() {
    if (!confirm(`Удалить пользователя «${username}» и все его конфиги? Это действие нельзя отменить.`)) return
    try { await api(`/api/admin/users/${encodeURIComponent(username)}`, { method: 'DELETE' }); navigate('/admin') } catch (err) { setError(err.message) }
  }
  const active = !data?.expires_at || new Date(data.expires_at) > new Date()
  return <Layout user={user} title={username} subtitle="Профиль пользователя и статистика подключений" action={<button class="button button-ghost" onClick={() => navigate('/admin')}><Icon name="back" /> К списку</button>}>
    <Notice>{error}</Notice>
    {!data ? !error && <Loader /> : <>
      <section class="profile-card"><div class="profile-main"><span class="profile-avatar">{username[0].toUpperCase()}</span><div><span class={`status ${active ? 'active' : 'expired'}`}>{active ? 'Подписка активна' : 'Подписка истекла'}</span><h2>{username}</h2><p>Создан {formatDate(data.created_at)}</p></div></div><div class="profile-expiry"><small>Доступ до</small><strong>{formatDate(data.expires_at)}</strong></div><div class="profile-actions"><button class="button button-secondary" onClick={() => setExtendOpen(true)}>Продлить</button><button class="button button-danger" onClick={remove}><Icon name="trash" /> Удалить</button></div></section>
      <section class="section-block"><div class="section-title"><div><h2>Конфиги и статистика</h2><p>{data.configs.length} подключений пользователя</p></div></div>
        {!data.configs.length ? <div class="empty compact"><h3>Конфигов нет</h3></div> : <div class="admin-configs">{data.configs.map((config) => <article class="admin-config" key={config.name}><div class="admin-config-head"><span class="device-icon"><Icon name="bolt" /></span><div><h3>{config.name}</h3><span>{config.ip || 'IP недоступен'}</span></div><button class="button button-ghost" onClick={() => navigate(`/subscription/${config.sub_token}`)}>Подписка <Icon name="arrow" /></button></div><div class="traffic"><div><small>Получено</small><strong>{formatBytes(config.received)}</strong></div><div><small>Отправлено</small><strong>{formatBytes(config.sent)}</strong></div><div><small>Последняя связь</small><strong>{config.last_handshake ? formatDate(config.last_handshake * 1000, true) : '—'}</strong></div></div></article>)}</div>}
      </section>
    </>}
    {extendOpen && <Modal title="Продлить подписку" onClose={() => setExtendOpen(false)}><p>Новая дата будет рассчитана от сегодняшнего дня.</p><form class="form-stack" onSubmit={extend}><label>Количество дней<input name="days" type="number" min="1" defaultValue="30" required autoFocus /></label><button class="button button-primary button-wide">Продлить подписку</button></form></Modal>}
  </Layout>
}
