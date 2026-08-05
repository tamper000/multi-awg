import { useEffect, useState } from 'preact/hooks'
import { session } from './api.js'
import { Toast } from './ui.jsx'
import Login from './pages/Login.jsx'
import UserPanel from './pages/UserPanel.jsx'
import Subscription from './pages/Subscription.jsx'
import AdminDashboard from './pages/AdminDashboard.jsx'
import AdminUser from './pages/AdminUser.jsx'

export function navigate(path) {
  history.pushState({}, '', path)
  window.dispatchEvent(new Event('navigate'))
}

export default function App() {
  const [path, setPath] = useState(location.pathname)
  const [user, setUser] = useState(session.user())

  useEffect(() => {
    const updatePath = () => setPath(location.pathname)
    const expired = () => { setUser(null); navigate('/login') }
    addEventListener('popstate', updatePath)
    addEventListener('navigate', updatePath)
    addEventListener('session-expired', expired)
    return () => {
      removeEventListener('popstate', updatePath)
      removeEventListener('navigate', updatePath)
      removeEventListener('session-expired', expired)
    }
  }, [])

  const subscriptionToken = path.match(/^\/subscription\/([^/]+)$/)?.[1]
  if (subscriptionToken) return <><Subscription token={subscriptionToken} user={user} /><Toast /></>
  const loginLink = path === '/login' && new URLSearchParams(location.hash.slice(1)).has('password')
  if (loginLink) return <><Login onLogin={(nextUser) => setUser(nextUser)} /><Toast /></>
  if (!user || !session.token()) return <><Login onLogin={(nextUser) => setUser(nextUser)} /><Toast /></>

  if (user.role === 'admin') {
    const userID = path.match(/^\/admin\/users\/(\d+)$/)?.[1]
    if (userID) return <><AdminUser userID={userID} user={user} /><Toast /></>
    return <><AdminDashboard user={user} onLogout={() => setUser(null)} /><Toast /></>
  }
  return <><UserPanel user={user} onLogout={() => setUser(null)} /><Toast /></>
}
