import { useState, useEffect } from 'react'

interface Referral {
  id: string
  candidate_name: string
  linkedin_url: string | null
  role: string | null
  job_id: string | null
  referrer_name: string | null
  stage: string
  app_status: string
  created_at: string
  company: string | null
  current_title: string | null
}

const stageColors: Record<string, string> = {
  'Submitted': 'bg-gray-100 text-gray-700',
  'Application Review': 'bg-blue-100 text-blue-700',
  'Initial Phone Screen': 'bg-sky-100 text-sky-700',
  'Technical Phone Screen': 'bg-indigo-100 text-indigo-700',
  'Pre-Interview Sell Chat': 'bg-cyan-100 text-cyan-700',
  'Onsite and Leads Chat': 'bg-purple-100 text-purple-700',
  'Post-Interview Sell Chat': 'bg-fuchsia-100 text-fuchsia-700',
  'Leads Chat': 'bg-violet-100 text-violet-700',
  'Offer': 'bg-green-100 text-green-700',
  'Hired': 'bg-emerald-100 text-emerald-800',
  'Rejected': 'bg-red-100 text-red-700',
  'Archived': 'bg-orange-100 text-orange-700',
}

function getStageBadgeClass(stage: string): string {
  return stageColors[stage] || 'bg-gray-100 text-gray-700'
}

export default function ReferralDashboard({
  initialStageFilter,
  onFilterApplied,
}: {
  initialStageFilter: string
  onFilterApplied: () => void
}) {
  const [referrals, setReferrals] = useState<Referral[]>([])
  const [loading, setLoading] = useState(true)
  const [stageFilter, setStageFilter] = useState(initialStageFilter)
  const [roleFilter, setRoleFilter] = useState('')
  const [search, setSearch] = useState('')

  useEffect(() => {
    if (initialStageFilter) {
      setStageFilter(initialStageFilter)
      onFilterApplied()
    }
  }, [initialStageFilter, onFilterApplied])

  useEffect(() => {
    const fetchReferrals = async () => {
      setLoading(true)
      try {
        const params = new URLSearchParams()
        if (stageFilter) params.set('stage', stageFilter)
        if (roleFilter) params.set('role', roleFilter)
        const res = await fetch(`/api/referrals?${params}`)
        const data = await res.json()
        setReferrals(data.referrals || [])
      } catch {
        setReferrals([])
      } finally {
        setLoading(false)
      }
    }
    fetchReferrals()
  }, [stageFilter, roleFilter])

  const filtered = referrals.filter((r) => {
    if (!search) return true
    const q = search.toLowerCase()
    return (
      r.candidate_name.toLowerCase().includes(q) ||
      (r.role && r.role.toLowerCase().includes(q)) ||
      (r.referrer_name && r.referrer_name.toLowerCase().includes(q))
    )
  })

  const stages = [...new Set(referrals.map((r) => r.stage))].sort()

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Referrals</h2>
        <p className="text-sm text-gray-500 mt-1">
          {filtered.length} referral{filtered.length !== 1 ? 's' : ''} found
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-3 mb-6">
        <input
          type="text"
          placeholder="Search by name, role, or referrer..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="px-4 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent w-72"
        />
        <select
          value={stageFilter}
          onChange={(e) => setStageFilter(e.target.value)}
          className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500"
        >
          <option value="">All Stages</option>
          {stages.map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
        <input
          type="text"
          placeholder="Filter by role..."
          value={roleFilter}
          onChange={(e) => setRoleFilter(e.target.value)}
          className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 w-48"
        />
        {(stageFilter || roleFilter || search) && (
          <button
            onClick={() => { setStageFilter(''); setRoleFilter(''); setSearch('') }}
            className="px-3 py-2 text-sm text-gray-500 hover:text-gray-700"
          >
            Clear filters
          </button>
        )}
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        </div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-16 text-gray-400">
          <p className="text-lg">No referrals found</p>
          <p className="text-sm mt-1">SDS referrals from Ashby will appear here.</p>
        </div>
      ) : (
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="bg-gray-50 border-b border-gray-200">
                <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Candidate</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Role</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Referrer</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Stage</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Company</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Date</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {filtered.map((r) => (
                <tr key={r.id} className="hover:bg-gray-50 transition-colors">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-gray-900 text-sm">{r.candidate_name}</span>
                      {r.linkedin_url && (
                        <a
                          href={r.linkedin_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-blue-500 hover:text-blue-700"
                          title="LinkedIn Profile"
                        >
                          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                            <path d="M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433a2.062 2.062 0 01-2.063-2.065 2.064 2.064 0 112.063 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z"/>
                          </svg>
                        </a>
                      )}
                    </div>
                    {r.current_title && (
                      <p className="text-xs text-gray-400 mt-0.5">{r.current_title}</p>
                    )}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">{r.role || '—'}</td>
                  <td className="px-4 py-3 text-sm text-gray-600">{r.referrer_name || '—'}</td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStageBadgeClass(r.stage)}`}>
                      {r.stage}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500">{r.company || '—'}</td>
                  <td className="px-4 py-3 text-sm text-gray-500">
                    {new Date(r.created_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
