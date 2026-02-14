import axios from 'axios'

export const api = axios.create({
  baseURL: '/manage/api',
  withCredentials: true,
})

// CSRF: read token from cookie and send it as X-CSRF-Token header on every request
api.interceptors.request.use((config) => {
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]+)/)
  if (match) {
    config.headers['X-CSRF-Token'] = match[1]
  }
  return config
})
