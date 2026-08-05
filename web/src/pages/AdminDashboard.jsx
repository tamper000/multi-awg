import { useEffect, useState } from 'preact/hooks'
import { api, jsonBody } from '../api.js'
import { copyText, Empty, formatDate, Icon, Loader, Modal, Notice } from '../ui.jsx'
import { navigate } from '../App.jsx'
import Layout from './Layout.jsx'

export default function AdminDashboard({ user, onLogout }) {
  const [users, setUsers] = useState(null)
  const [error, setError] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [credentials, setCredentials] = useState(null)
  const [copied, setCopied] = useState(false)
  const [pwOpen, setPwOpen] = useState(false)
  const [pwDone, setPwDone] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [syncMsg, setSyncMsg] = useState('')
  const [syncKind, setSyncKind] = useState('success')

  const loginLink = credentials && `${location.origin}/login#${new URLSearchParams({ username: credentials.username, password: credentials.password })}`

  async function load() {
    try { setUsers(await api('/api/admin/users')); setError('') } catch (err) { setError(err.message) }
  }
  useEffect(() => { load() }, [])

  async function create(event) {
    event.preventDefault()
    const form = new FormData(event.currentTarget)
    try {
      const result = await api('/api/admin/users', { method: 'POST', body: jsonBody({ username: form.get('username'), days: Number(form.get('days')) }) })
      setCreateOpen(false); setCredentials(result); load()
    } catch {}
  }

  async function changePassword(event) {
    event.preventDefault()
    const form = new FormData(event.currentTarget)
    setPwDone(false)
    try {
      await api('/api/user/password', { method: 'POST', body: jsonBody({ old: form.get('old'), new: form.get('new') }) })
      setPwDone(true)
    } catch {}
  }

  async function sync() {
    setSyncing(true); setSyncMsg('')
    try {
      await api('/api/admin/sync', { method: 'POST' })
      setSyncKind('success'); setSyncMsg('Синхронизация завершена')
    } catch (err) { setSyncKind('error'); setSyncMsg(err.message) }
    setTimeout(() => setSyncMsg(''), 3000)
    setSyncing(false)
  }

  const clients = users?.filter((item) => item.role === 'user') || []
  const active = clients.filter((item) => !item.expires_at || new Date(item.expires_at) > new Date()).length
  return <Layout user={user} onLogout={onLogout} onSync={sync} syncing={syncing} title="Пользователи" subtitle="Управление доступом и подписками" action={<button class="button button-primary" onClick={() => setCreateOpen(true)}><Icon name="plus" /> Добавить пользователя</button>}>
    <Notice>{error}</Notice>{syncMsg && <div class="sync-notice"><Notice kind={syncKind}>{syncMsg}</Notice></div>}
    {!users ? <Loader /> : <>
      <section class="stats-grid admin-stats"><article class="stat-card accent-lime"><small>Всего пользователей</small><strong>{clients.length}</strong><span>Зарегистрировано</span></article><article class="stat-card accent-purple"><small>Активные подписки</small><strong>{active}</strong><span>Доступ разрешён</span></article><article class="stat-card"><small>Истекли</small><strong>{clients.length - active}</strong><span>Нужно продление</span></article></section>
      <section class="section-block"><div class="section-title"><div><h2>Список пользователей</h2><p>Откройте профиль для статистики и управления</p></div><button class="button button-ghost" onClick={() => setPwOpen(true)}><Icon name="key" /> Сменить пароль</button></div>
        {!clients.length ? <Empty title="Пользователей пока нет" text="Добавьте первого пользователя и передайте ему созданный пароль." /> : <div class="user-table"><div class="table-row table-head"><span>Пользователь</span><span>Статус</span><span>Подписка до</span><span>Создан</span><span /></div>{clients.map((item) => {
          const isActive = !item.expires_at || new Date(item.expires_at) > new Date()
          return <button class="table-row" onClick={() => navigate(`/admin/users/${item.id}`)} key={item.id}><span class="user-cell"><b>{item.username[0].toUpperCase()}</b><strong>{item.username}</strong></span><span><i class={`status ${isActive ? 'active' : 'expired'}`}>{isActive ? 'Активна' : 'Истекла'}</i></span><span>{formatDate(item.expires_at)}</span><span>{formatDate(item.created_at)}</span><span class="row-arrow"><Icon name="arrow" /></span></button>
        })}</div>}
      </section>
    </>}
    {createOpen && <Modal title="Новый пользователь" onClose={() => setCreateOpen(false)}><p>Пароль будет создан автоматически и показан один раз.</p><form class="form-stack" onSubmit={create}><label>Логин<input name="username" required pattern="[A-Za-zА-Яа-яЁё0-9]+" placeholder="Например, alex" autoFocus /></label><label>Срок подписки, дней<input name="days" type="number" min="1" defaultValue="30" required /></label><button class="button button-primary button-wide"><Icon name="plus" /> Создать пользователя</button></form></Modal>}
    {credentials && <Modal title="Пользователь создан" onClose={() => setCredentials(null)}><Notice kind="warning">Пароль и ссылка показываются только сейчас. Передайте их пользователю.</Notice><div class="credential"><small>Логин</small><strong>{credentials.username}</strong></div><div class="credential"><small>Пароль</small><strong>{credentials.password}</strong><button class="icon-button" onClick={() => copyText(credentials.password, setCopied)} aria-label="Скопировать пароль"><Icon name="copy" /></button></div><div class="credential credential-link"><small>Ссылка для автоматического входа</small><strong>{loginLink}</strong><button class="icon-button" onClick={() => copyText(loginLink, setCopied)} aria-label="Скопировать ссылку"><Icon name="copy" /></button></div><button class="button button-primary button-wide" onClick={() => setCredentials(null)}>{copied ? 'Скопировано' : 'Готово'}</button></Modal>}
    {pwOpen && <Modal title="Смена пароля" onClose={() => { setPwOpen(false); setPwDone(false) }}>{pwDone ? <p class="pw-done">Пароль изменён.</p> : <form class="form-stack" onSubmit={changePassword}><label>Текущий пароль<input type="password" name="old" required autoFocus /></label><label>Новый пароль<input type="password" name="new" required /></label><button class="button button-primary button-wide">Сохранить пароль</button></form>}</Modal>}
  </Layout>
}
