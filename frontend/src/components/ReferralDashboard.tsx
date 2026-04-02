import { useState, useEffect, useCallback } from 'react'

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

const stageSelectedColors: Record<string, string> = {
  'Submitted': 'bg-gray-600 text-white',
  'Application Review': 'bg-blue-600 text-white',
  'Initial Phone Screen': 'bg-sky-600 text-white',
  'Technical Phone Screen': 'bg-indigo-600 text-white',
  'Pre-Interview Sell Chat': 'bg-cyan-600 text-white',
  'Onsite and Leads Chat': 'bg-purple-600 text-white',
  'Post-Interview Sell Chat': 'bg-fuchsia-600 text-white',
  'Leads Chat': 'bg-violet-600 text-white',
  'Offer': 'bg-green-600 text-white',
  'Hired': 'bg-emerald-600 text-white',
  'Rejected': 'bg-red-600 text-white',
  'Archived': 'bg-orange-600 text-white',
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
  const [allReferrals, setAllReferrals] = useState<Referral[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedStages, setSelectedStages] = useState<Set<string>>(
    () => initialStageFilter ? new Set([initialStageFilter]) : new Set(),
  )
  const [roleFilter, setRoleFilter] = useState('')
  const [search, setSearch] = useState('')

  useEffect(() => {
    if (initialStageFilter) {
      setSelectedStages(new Set([initialStageFilter]))
      onFilterApplied()
    }
  }, [initialStageFilter, onFilterApplied])

  useEffect(() => {
    const fetchReferrals = async () => {
      setLoading(true)
      try {
        const params = new URLSearchParams()
        if (roleFilter) params.set('role', roleFilter)
        const res = await fetch(`/api/referrals?${params}`)
        const data = await res.json()
        setAllReferrals(data.referrals || [])
      } catch {
        setAllReferrals([])
      } finally {
        setLoading(false)
      }
    }
    fetchReferrals()
  }, [roleFilter])

  const toggleStage = useCallback((stage: string) => {
    setSelectedStages((prev) => {
      const next = new Set(prev)
      if (next.has(stage)) {
        next.delete(stage)
      } else {
        next.add(stage)
      }
      return next
    })
  }, [])

  const stages = [...new Set(allReferrals.map((r) => r.stage))].sort()

  const filtered = allReferrals.filter((r) => {
    if (selectedStages.size > 0 && !selectedStages.has(r.stage)) return false
    if (!search) return true
    const q = search.toLowerCase()
    return (
      r.candidate_name.toLowerCase().includes(q) ||
      (r.role && r.role.toLowerCase().includes(q)) ||
      (r.referrer_name && r.referrer_name.toLowerCase().includes(q))
    )
  })

  const hasFilters = selectedStages.size > 0 || roleFilter || search

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Referrals</h2>
        <p className="text-sm text-gray-500 mt-1">
          {filtered.length} referral{filtered.length !== 1 ? 's' : ''} found
          {selectedStages.size > 0 && ` in ${selectedStages.size} stage${selectedStages.size !== 1 ? 's' : ''}`}
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-3 mb-4">
        <input
          type="text"
          placeholder="Search by name, role, or referrer..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="px-4 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent w-72"
        />
        <input
          type="text"
          placeholder="Filter by role..."
          value={roleFilter}
          onChange={(e) => setRoleFilter(e.target.value)}
          className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 w-48"
        />
        {hasFilters && (
          <button
            onClick={() => { setSelectedStages(new Set()); setRoleFilter(''); setSearch('') }}
            className="px-3 py-2 text-sm text-gray-500 hover:text-gray-700"
          >
            Clear filters
          </button>
        )}
      </div>

      {!loading && stages.length > 0 && (
        <div className="flex flex-wrap gap-2 mb-6">
          {stages.map((stage) => {
            const isSelected = selectedStages.has(stage)
            const colorClass = isSelected
              ? (stageSelectedColors[stage] || 'bg-gray-600 text-white')
              : (stageColors[stage] || 'bg-gray-100 text-gray-700')
            const count = allReferrals.filter((r) => r.stage === stage).length
            return (
              <button
                key={stage}
                onClick={() => toggleStage(stage)}
                className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium transition-all ${colorClass} ${
                  isSelected ? 'ring-2 ring-offset-1 ring-gray-400' : 'hover:opacity-80'
                }`}
              >
                {stage}
                <span className={`${isSelected ? 'opacity-80' : 'opacity-60'}`}>({count})</span>
              </button>
            )
          })}
        </div>
      )}

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
                    {r.linkedin_url ? (
                      <a
                        href={r.linkedin_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="font-medium text-blue-600 hover:text-blue-800 hover:underline text-sm"
                      >
                        {r.candidate_name}
                      </a>
                    ) : (
                      <span className="font-medium text-gray-900 text-sm">{r.candidate_name}</span>
                    )}
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
