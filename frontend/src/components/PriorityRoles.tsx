import { useState, useEffect } from 'react'

interface Job {
  greenhouse_id: number
  title: string
  department: string | null
  team: string | null
  status: string
  is_priority: boolean
  referral_count: number
  location: string | null
  last_synced_at: string | null
}

export default function PriorityRoles() {
  const [jobs, setJobs] = useState<Job[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [showPriorityOnly, setShowPriorityOnly] = useState(false)

  const fetchJobs = async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams({ status: 'open' })
      if (showPriorityOnly) params.set('priority', 'true')
      const res = await fetch(`/api/jobs?${params}`)
      const data = await res.json()
      setJobs(data.jobs || [])
    } catch {
      setJobs([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchJobs() }, [showPriorityOnly])

  const togglePriority = async (ghID: number, current: boolean) => {
    try {
      await fetch(`/api/jobs/${ghID}/priority`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_priority: !current }),
      })
      setJobs((prev) =>
        prev.map((j) => j.greenhouse_id === ghID ? { ...j, is_priority: !current } : j)
      )
    } catch { /* ignore */ }
  }

  const filtered = jobs.filter((j) => {
    if (!search) return true
    const q = search.toLowerCase()
    return (
      j.title.toLowerCase().includes(q) ||
      (j.department && j.department.toLowerCase().includes(q)) ||
      (j.location && j.location.toLowerCase().includes(q))
    )
  })

  const priorityCount = jobs.filter((j) => j.is_priority).length
  const totalReferrals = jobs.reduce((sum, j) => sum + j.referral_count, 0)

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Priority Roles</h2>
        <p className="text-sm text-gray-500 mt-1">
          {jobs.length} open position{jobs.length !== 1 ? 's' : ''} &middot; {priorityCount} priority &middot; {totalReferrals} total referrals
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-3 mb-6">
        <input
          type="text"
          placeholder="Search roles..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="px-4 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent w-72"
        />
        <label className="flex items-center gap-2 text-sm text-gray-600 cursor-pointer">
          <input
            type="checkbox"
            checked={showPriorityOnly}
            onChange={(e) => setShowPriorityOnly(e.target.checked)}
            className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
          />
          Priority only
        </label>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        </div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-16 text-gray-400">
          <p className="text-lg">No roles found</p>
          <p className="text-sm mt-1">Open positions will appear after syncing from Greenhouse.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filtered.map((job) => (
            <div
              key={job.greenhouse_id}
              className={`bg-white rounded-xl border p-5 transition-shadow hover:shadow-md ${
                job.is_priority ? 'border-amber-300 ring-1 ring-amber-200' : 'border-gray-200'
              }`}
            >
              <div className="flex items-start justify-between mb-3">
                <div className="flex-1 min-w-0">
                  <h3 className="font-semibold text-gray-900 text-sm truncate" title={job.title}>
                    {job.title}
                  </h3>
                  <p className="text-xs text-gray-500 mt-0.5">{job.department || 'No department'}</p>
                </div>
                <button
                  onClick={() => togglePriority(job.greenhouse_id, job.is_priority)}
                  className={`ml-2 flex-shrink-0 p-1.5 rounded-lg transition-colors ${
                    job.is_priority
                      ? 'bg-amber-100 text-amber-600 hover:bg-amber-200'
                      : 'bg-gray-100 text-gray-400 hover:bg-gray-200 hover:text-gray-600'
                  }`}
                  title={job.is_priority ? 'Remove priority' : 'Mark as priority'}
                >
                  <svg className="w-4 h-4" fill={job.is_priority ? 'currentColor' : 'none'} stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                  </svg>
                </button>
              </div>

              {job.location && (
                <p className="text-xs text-gray-400 mb-3">{job.location}</p>
              )}

              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5">
                  <span className="text-lg font-bold text-gray-900">{job.referral_count}</span>
                  <span className="text-xs text-gray-500">referral{job.referral_count !== 1 ? 's' : ''}</span>
                </div>
                {job.is_priority && (
                  <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-700">
                    Priority
                  </span>
                )}
              </div>

              <div className="mt-3 pt-3 border-t border-gray-100">
                <a
                  href={`https://app.greenhouse.io/sdash/jobs/${job.greenhouse_id}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-xs text-blue-500 hover:text-blue-700 font-medium"
                >
                  View in Greenhouse →
                </a>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
