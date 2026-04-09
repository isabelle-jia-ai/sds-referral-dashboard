import { useState, useEffect, type ReactNode } from 'react'

const TOKEN_KEY = 'protected_tab_token'

export default function ProtectedTab({ children }: { children: ReactNode }) {
  const [unlocked, setUnlocked] = useState(false)
  const [checking, setChecking] = useState(true)
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    const token = sessionStorage.getItem(TOKEN_KEY)
    if (!token) {
      setChecking(false)
      return
    }
    fetch('/api/auth/check', { headers: { 'X-Auth-Token': token } })
      .then((r) => r.json())
      .then((data) => {
        if (data.valid) {
          setUnlocked(true)
        } else {
          sessionStorage.removeItem(TOKEN_KEY)
        }
      })
      .catch(() => sessionStorage.removeItem(TOKEN_KEY))
      .finally(() => setChecking(false))
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSubmitting(true)

    try {
      const resp = await fetch('/api/auth/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      })
      const data = await resp.json()

      if (resp.ok && data.token) {
        sessionStorage.setItem(TOKEN_KEY, data.token)
        setUnlocked(true)
      } else {
        setError(data.error || 'Incorrect password')
      }
    } catch {
      setError('Unable to verify. Please try again.')
    } finally {
      setSubmitting(false)
    }
  }

  if (checking) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    )
  }

  if (unlocked) {
    return <>{children}</>
  }

  return (
    <div className="flex items-center justify-center h-full bg-gray-50">
      <form onSubmit={handleSubmit} className="bg-white rounded-xl shadow-sm border border-gray-200 p-8 w-full max-w-sm">
        <div className="text-center mb-6">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-blue-50 mb-3">
            <span className="text-2xl">🔒</span>
          </div>
          <h2 className="text-lg font-semibold text-gray-900">Protected Section</h2>
          <p className="text-sm text-gray-500 mt-1">Enter the password to access this area.</p>
        </div>

        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Password"
          autoFocus
          className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />

        {error && (
          <p className="text-sm text-red-600 mt-2">{error}</p>
        )}

        <button
          type="submit"
          disabled={submitting || !password}
          className="w-full mt-4 px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {submitting ? 'Verifying...' : 'Unlock'}
        </button>
      </form>
    </div>
  )
}
