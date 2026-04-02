import { useState, useEffect } from 'react'

interface Job {
  id: string
  title: string
  department: string | null
  status: string
  referral_count: number
  location: string | null
  job_url: string | null
}

export default function PriorityRoles() {
  const [jobs, setJobs] = useState<Job[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')

  useEffect(() => {
    const fetchJobs = async () => {
      setLoading(true)
      try {
        const res = await fetch('/api/jobs')
        const data = await res.json()
        setJobs(data.jobs || [])
      } catch {
        setJobs([])
      } finally {
        setLoading(false)
      }
    }
    fetchJobs()
  }, [])

  const filtered = jobs.filter((j) => {
    if (!search) return true
    const q = search.toLowerCase()
    return (
      j.title.toLowerCase().includes(q) ||
      (j.department && j.department.toLowerCase().includes(q)) ||
      (j.location && j.location.toLowerCase().includes(q))
    )
  })

  const totalReferrals = jobs.reduce((sum, j) => sum + j.referral_count, 0)

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Open Roles</h2>
        <p className="text-sm text-gray-500 mt-1">
          {jobs.length} open position{jobs.length !== 1 ? 's' : ''} &middot; {totalReferrals} total referrals
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
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        </div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-16 text-gray-400">
          <p className="text-lg">No roles found</p>
          <p className="text-sm mt-1">Open SDS positions will appear here.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filtered.map((job) => (
            <a
              key={job.id}
              href={job.job_url || 'https://www.applied.co/careers'}
              target="_blank"
              rel="noopener noreferrer"
              className="bg-white rounded-xl border border-gray-200 p-5 transition-all hover:shadow-md hover:border-blue-300 group block"
            >
              <div className="mb-3">
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold text-gray-900 text-sm truncate group-hover:text-blue-600 transition-colors" title={job.title}>
                    {job.title}
                  </h3>
                  <svg className="w-4 h-4 text-gray-300 group-hover:text-blue-500 flex-shrink-0 ml-2 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                  </svg>
                </div>
                {job.department && (
                  <p className="text-xs text-gray-500 mt-0.5">{job.department}</p>
                )}
              </div>

              {job.location && (
                <p className="text-xs text-gray-400 mb-3">{job.location}</p>
              )}

              <div className="flex items-center gap-1.5">
                <span className="text-lg font-bold text-gray-900">{job.referral_count}</span>
                <span className="text-xs text-gray-500">referral{job.referral_count !== 1 ? 's' : ''}</span>
              </div>
            </a>
          ))}
        </div>
      )}
    </div>
  )
}
