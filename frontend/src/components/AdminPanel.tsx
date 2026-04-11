import { useState, useEffect } from 'react'

interface HiredByRole {
  role: string
  total_hires: number
  referral_hires: number
}

interface DeptComparison {
  department: string
  total: number
  quarters: Record<string, number>
}

interface HiredCandidate {
  candidate_name: string
  role: string
  year: number
  created_at: string
  gh_profile_url: string
}

export default function AdminPanel() {
  const [hiredByRole, setHiredByRole] = useState<HiredByRole[]>([])
  const [deptComparison, setDeptComparison] = useState<DeptComparison[]>([])
  const [hiredList, setHiredList] = useState<HiredCandidate[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchAll = async () => {
      setLoading(true)
      try {
        const [hiredRoleRes, compRes, hiredListRes] = await Promise.all([
          fetch('/api/referrals/hired-by-role'),
          fetch('/api/referrals/company-comparison'),
          fetch('/api/referrals/hired-list'),
        ])
        const [hiredRoleData, compData, hiredListData] = await Promise.all([
          hiredRoleRes.json(),
          compRes.json(),
          hiredListRes.json(),
        ])
        setHiredByRole(hiredRoleData.roles || [])
        setDeptComparison(compData.departments || [])
        setHiredList(hiredListData.hires || [])
      } catch { /* ignore */ } finally {
        setLoading(false)
      }
    }
    fetchAll()
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  const allQLabels = Array.from(
    new Set(deptComparison.flatMap((d) => Object.keys(d.quarters)))
  ).sort()

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-8">
      <div>
        <h2 className="text-2xl font-bold text-gray-900">Hiring Manager Analytics</h2>
        <p className="text-sm text-gray-500 mt-1">Protected hiring data and cross-org comparison</p>
      </div>

      {/* All Referral Hires by Year */}
      {hiredList.length > 0 && (() => {
        const years = Array.from(new Set(hiredList.map((h) => h.year))).sort((a, b) => b - a)
        const byYear = new Map<number, HiredCandidate[]>()
        for (const h of hiredList) {
          if (!byYear.has(h.year)) byYear.set(h.year, [])
          byYear.get(h.year)!.push(h)
        }
        return (
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <div className="flex items-center justify-between mb-1">
              <h3 className="text-lg font-semibold text-gray-900">All Referral Hires</h3>
              <span className="text-sm font-semibold text-emerald-700 bg-emerald-50 px-3 py-1 rounded-full">
                {hiredList.length} total
              </span>
            </div>
            <p className="text-xs text-gray-400 mb-5">Every candidate hired through a referral, by year</p>
            <div className={`grid gap-6 ${years.length >= 3 ? 'grid-cols-1 lg:grid-cols-3' : years.length === 2 ? 'grid-cols-1 lg:grid-cols-2' : 'grid-cols-1'}`}>
              {years.map((year) => {
                const candidates = byYear.get(year) || []
                return (
                  <div key={year}>
                    <div className="flex items-center gap-2 mb-3 pb-2 border-b border-gray-100">
                      <span className="text-base font-bold text-gray-900">{year}</span>
                      <span className="text-xs text-gray-400">({candidates.length} hires)</span>
                    </div>
                    <div className="space-y-2.5">
                      {candidates.map((h, i) => (
                        <div key={`${h.candidate_name}-${i}`} className="flex items-start gap-2">
                          <span className="text-emerald-500 mt-0.5 text-xs">&#9679;</span>
                          <div className="min-w-0">
                            {h.gh_profile_url ? (
                              <a
                                href={h.gh_profile_url}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-sm font-medium text-blue-600 hover:text-blue-800 hover:underline"
                              >
                                {h.candidate_name}
                              </a>
                            ) : (
                              <span className="text-sm font-medium text-gray-900">{h.candidate_name}</span>
                            )}
                            <p className="text-xs text-gray-400 truncate" title={h.role}>{h.role}</p>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )
      })()}

      {/* Referral-to-Hire Rate by Role */}
      {hiredByRole.length > 0 && (() => {
        const totalH = hiredByRole.reduce((s, r) => s + r.total_hires, 0)
        const totalRH = hiredByRole.reduce((s, r) => s + r.referral_hires, 0)
        const overallPct = totalH > 0 ? Math.round((totalRH / totalH) * 100) : 0
        const maxHires = Math.max(...hiredByRole.map((r) => r.total_hires), 1)
        return (
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <div className="flex items-center justify-between mb-1">
              <h3 className="text-lg font-semibold text-gray-900">Referral-to-Hire Rate by Role</h3>
              <span className="text-sm font-semibold text-blue-700 bg-blue-50 px-3 py-1 rounded-full">
                {overallPct}% overall ({totalRH}/{totalH})
              </span>
            </div>
            <p className="text-xs text-gray-400 mb-5">All SDS hires: referral share shown in blue</p>
            <div className="space-y-3 max-h-[480px] overflow-y-auto">
              {[...hiredByRole].sort((a, b) => {
                const pctA = a.total_hires > 0 ? a.referral_hires / a.total_hires : 0
                const pctB = b.total_hires > 0 ? b.referral_hires / b.total_hires : 0
                return pctB - pctA || b.total_hires - a.total_hires
              }).map((r) => {
                const pct = r.total_hires > 0 ? Math.round((r.referral_hires / r.total_hires) * 100) : 0
                const barWidth = (r.total_hires / maxHires) * 100
                const refShare = r.total_hires > 0 ? (r.referral_hires / r.total_hires) * 100 : 0
                return (
                  <div key={r.role}>
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-sm text-gray-700 truncate max-w-[55%]" title={r.role}>
                        {r.role}
                      </span>
                      <span className="text-xs text-gray-500 ml-2 whitespace-nowrap">
                        <span className={`font-semibold ${pct >= 25 ? 'text-blue-700' : 'text-gray-700'}`}>{pct}%</span>
                        {' '}({r.referral_hires}/{r.total_hires} hires)
                      </span>
                    </div>
                    <div className="w-full bg-gray-100 rounded-full h-2.5" style={{ width: `${Math.max(barWidth, 8)}%` }}>
                      <div
                        className="bg-blue-500 h-2.5 rounded-full transition-all"
                        style={{ width: `${refShare}%`, minWidth: r.referral_hires > 0 ? '4px' : '0' }}
                      />
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )
      })()}

      {/* Company-wide comparison */}
      {deptComparison.length > 0 && (
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-1">Referrals vs. Other Organizations</h3>
          <p className="text-xs text-gray-400 mb-4">Top 10 departments by total referrals since 2025</p>
          <div className="space-y-3">
            {deptComparison.map((dept) => {
              const isSDS = dept.department === 'SDS (Combined)'
              const maxTotal = deptComparison[0]?.total || 1
              return (
                <div key={dept.department}>
                  <div className="flex items-center justify-between mb-1">
                    <span className={`text-sm font-medium ${isSDS ? 'text-blue-700' : 'text-gray-700'}`}>
                      {dept.department}
                      {isSDS && <span className="ml-1.5 text-[10px] bg-blue-100 text-blue-600 px-1.5 py-0.5 rounded-full font-semibold">YOU</span>}
                    </span>
                    <span className="text-sm font-semibold text-gray-900">{dept.total}</span>
                  </div>
                  <div className="w-full bg-gray-100 rounded-full h-2.5">
                    <div
                      className={`h-2.5 rounded-full transition-all ${isSDS ? 'bg-blue-500' : 'bg-gray-300'}`}
                      style={{ width: `${(dept.total / maxTotal) * 100}%` }}
                    />
                  </div>
                  <div className="flex gap-1 mt-1">
                    {allQLabels.map((q) => {
                      const count = dept.quarters[q] || 0
                      return (
                        <span key={q} className="text-[9px] text-gray-400 flex-1 text-center">
                          {q.replace(/^\d{4}-/, '')}: {count}
                        </span>
                      )
                    })}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
