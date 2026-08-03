const TOKEN_KEY = 'banana_token'
const USER_KEY = 'banana_user'

export const session = {
  token: () => localStorage.getItem(TOKEN_KEY),
  user: () => {
    try {
      return JSON.parse(localStorage.getItem(USER_KEY))
    } catch {
      return null
    }
  },
  set(token, user) {
    localStorage.setItem(TOKEN_KEY, token)
    localStorage.setItem(USER_KEY, JSON.stringify(user))
  },
  clear() {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  },
}

export async function api(path, options = {}) {
  const headers = { ...options.headers }
  const token = session.token()
  if (token) headers.Authorization = `Bearer ${token}`
  if (options.body) headers['Content-Type'] = 'application/json'

  let response
  try {
    response = await fetch(path, { ...options, headers })
  } catch {
    throw new Error('Сервер недоступен. Попробуйте ещё раз.')
  }

  const type = response.headers.get('content-type') || ''
  const body = type.includes('application/json') ? await response.json() : await response.text()
  if (response.status === 401 && path !== '/api/auth/login' && path !== '/api/user/password') {
    session.clear()
    window.dispatchEvent(new Event('session-expired'))
  }
  if (!response.ok) throw new Error(body?.error || `Ошибка ${response.status}`)
  return body
}

export const jsonBody = (value) => JSON.stringify(value)
