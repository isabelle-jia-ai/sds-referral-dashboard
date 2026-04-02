import { useState, useEffect } from 'react'

interface Stats {
  total_referrals: number
  active: number
  rejected: number
  open_jobs: number
}

interface StageCount {
  stage: string
  count: number
}

interface RoleCount {
  role: string
  count: number
}

interface QuarterlyCount {
  quarter: string
  count: number
}

export default function Analytics({ onStageClick }: { onStageClick: (stage: string) => void }) {
  const [stats, setStats] = useState<Stats | null>(null)
  const [stages, setStages] = useState<StageCount[]>([])
  const [roles, setRoles] = useState<RoleCount[]>([])
  const [quarters, setQuarters] = useState<QuarterlyCount[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchAll = async () => {
      setLoading(true)
      try {
        const [statsRes, stagesRes, rolesRes, quartersRes] = await Promise.all([
          fetch('/api/referrals/stats'),
          fetch('/api/referrals/by-stage'),
          fetch('/api/referrals/by-role'),
          fetch('/api/referrals/quarterly'),
        ])
        const [statsData, stagesData, rolesData, quartersData] = await Promise.all([
          statsRes.json(),
          stagesRes.json(),
          rolesRes.json(),
          quartersRes.json(),
        ])
        setStats(statsData.stats || null)
        setStages(stagesData.stages || [])
        setRoles(rolesData.roles || [])
        setQuarters(quartersData.quarters || [])
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

  const maxStageCount = Math.max(...stages.map((s) => s.count), 1)
  const maxRoleCount = Math.max(...roles.map((r) => r.count), 1)
  const maxQuarterCount = Math.max(...quarters.map((q) => q.count), 1)

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-8">
      <div>
        <h2 className="text-2xl font-bold text-gray-900">Analytics</h2>
        <p className="text-sm text-gray-500 mt-1">Referral pipeline overview</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <SummaryCard label="Total Referrals" value={stats?.total_referrals ?? 0} color="blue" />
        <SummaryCard label="Active" value={stats?.active ?? 0} color="green" />
        <SummaryCard label="Rejected" value={stats?.rejected ?? 0} color="red" />
        <SummaryCard label="Open Positions" value={stats?.open_jobs ?? 0} color="purple" />
      </div>

      <div className="bg-white rounded-xl border border-gray-200 p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Referrals by Stage</h3>
        {stages.length === 0 ? (
          <p className="text-gray-400 text-sm">No data yet</p>
        ) : (
          <div className="space-y-3">
            {stages.map((s) => (
              <button
                key={s.stage}
                onClick={() => onStageClick(s.stage)}
                className="w-full text-left group"
              >
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm font-medium text-gray-700 group-hover:text-blue-600 transition-colors">
                    {s.stage}
                  </span>
                  <span className="text-sm font-semibold text-gray-900">{s.count}</span>
                </div>
                <div className="w-full bg-gray-100 rounded-full h-2.5">
                  <div
                    className="bg-blue-500 h-2.5 rounded-full transition-all"
                    style={{ width: `${(s.count / maxStageCount) * 100}%` }}
                  />
                </div>
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Referrals by Role</h3>
          {roles.length === 0 ? (
            <p className="text-gray-400 text-sm">No data yet</p>
          ) : (
            <div className="space-y-3 max-h-80 overflow-y-auto">
              {roles.slice(0, 15).map((r) => (
                <div key={r.role}>
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-sm text-gray-700 truncate max-w-[200px]" title={r.role}>
                      {r.role}
                    </span>
                    <span className="text-sm font-semibold text-gray-900 ml-2">{r.count}</span>
                  </div>
                  <div className="w-full bg-gray-100 rounded-full h-2">
                    <div
                      className="bg-indigo-400 h-2 rounded-full transition-all"
                      style={{ width: `${(r.count / maxRoleCount) * 100}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Quarterly Trends</h3>
          {quarters.length === 0 ? (
            <p className="text-gray-400 text-sm">No data yet</p>
          ) : (
            <div className="flex items-end gap-2 h-48">
              {quarters.map((q) => (
                <div key={q.quarter} className="flex-1 flex flex-col items-center gap-1">
                  <span className="text-xs font-medium text-gray-700">{q.count}</span>
                  <div
                    className="w-full bg-blue-400 rounded-t-md transition-all min-h-[4px]"
                    style={{ height: `${(q.count / maxQuarterCount) * 100}%` }}
                  />
                  <span className="text-[10px] text-gray-400 whitespace-nowrap">
                    {q.quarter}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function SummaryCard({ label, value, color }: { label: string; value: number; color: string }) {
  const colorMap: Record<string, string> = {
    blue: 'bg-blue-50 text-blue-700 border-blue-200',
    green: 'bg-green-50 text-green-700 border-green-200',
    red: 'bg-red-50 text-red-700 border-red-200',
    purple: 'bg-purple-50 text-purple-700 border-purple-200',
  }

  return (
    <div className={`rounded-xl border p-5 ${colorMap[color] || colorMap.blue}`}>
      <p className="text-sm font-medium opacity-80">{label}</p>
      <p className="text-3xl font-bold mt-1">{value}</p>
    </div>
  )
}

