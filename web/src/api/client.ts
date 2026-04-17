import axios, { AxiosError } from 'axios'

const client = axios.create({
  baseURL: '/api/v1',
  withCredentials: true,
  headers: { 'Content-Type': 'application/json' },
})

let on401: (() => void) | null = null

export function setOn401Handler(fn: () => void) {
  on401 = fn
}

client.interceptors.response.use(
  (res) => res,
  (err: AxiosError) => {
    if (err.response?.status === 401 && on401) {
      on401()
    }
    return Promise.reject(err)
  },
)

export default client
