import { useState, useEffect } from 'react'

interface Stats {
  total_referrals: number
  active: number
  rejected: number
  hired: number
  open_jobs: number
}

interface QuarterlyCount {
  quarter: string
  count: number
}

interface HiredQuarter {
  quarter: string
  hired: number
}

interface LeaderboardEntry {
  referrer_name: string
  referral_count: number
}

export default function Analytics({ onStageClick }: { onStageClick: (stage: string) => void }) {
  const [stats, setStats] = useState<Stats | null>(null)
  const [quarters, setQuarters] = useState<QuarterlyCount[]>([])
  const [hiredQuarters, setHiredQuarters] = useState<HiredQuarter[]>([])
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchAll = async () => {
      setLoading(true)
      try {
        const [statsRes, quartersRes, hiredRes, lbRes] = await Promise.all([
          fetch('/api/referrals/stats'),
          fetch('/api/referrals/quarterly'),
          fetch('/api/referrals/hired-quarterly'),
          fetch('/api/referrals/leaderboard'),
        ])
        const [statsData, quartersData, hiredData, lbData] = await Promise.all([
          statsRes.json(),
          quartersRes.json(),
          hiredRes.json(),
          lbRes.json(),
        ])
        setStats(statsData.stats || null)
        setQuarters(quartersData.quarters || [])
        setHiredQuarters(hiredData.quarters || [])
        setLeaderboard(lbData.leaderboard || [])
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

  const maxQuarterCount = Math.max(...quarters.map((q) => q.count), 1)
  const maxHiredCount = Math.max(...hiredQuarters.map((q) => q.hired), 1)

  const allQLabels = Array.from(
    new Set([...quarters.map((q) => q.quarter), ...hiredQuarters.map((q) => q.quarter)])
  ).sort()

  const hiredMap = Object.fromEntries(hiredQuarters.map((q) => [q.quarter, q.hired]))
  const referralMap = Object.fromEntries(quarters.map((q) => [q.quarter, q.count]))

  const medalIcons = ['🥇', '🥈', '🥉']

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-8">
      <div>
        <h2 className="text-2xl font-bold text-gray-900">Analytics</h2>
        <p className="text-sm text-gray-500 mt-1">Referral pipeline overview</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-5 gap-4">
        <SummaryCard label="Total Referrals" value={stats?.total_referrals ?? 0} color="blue" />
        <SummaryCard label="Active" value={stats?.active ?? 0} color="green" onClick={() => onStageClick('Active')} />
        <SummaryCard label="Hired" value={stats?.hired ?? 0} color="emerald" onClick={() => onStageClick('Hired')} />
        <SummaryCard label="Rejected" value={stats?.rejected ?? 0} color="red" onClick={() => onStageClick('Rejected')} />
        <SummaryCard label="Open Positions" value={stats?.open_jobs ?? 0} color="purple" />
      </div>

      {/* Referrer Leaderboard */}
      {leaderboard.length > 0 && (
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center justify-between mb-1">
            <h3 className="text-lg font-semibold text-gray-900">Top Referrers</h3>
            <span className="text-xs text-gray-400">Since 2025</span>
          </div>
          <p className="text-xs text-gray-400 mb-5">People who have referred the most candidates</p>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-2">
            {leaderboard.slice(0, 10).map((entry, i) => {
              const maxCount = leaderboard[0]?.referral_count || 1
              return (
                <div key={entry.referrer_name} className="flex items-center gap-3 py-2">
                  <span className="w-8 text-center text-lg flex-shrink-0">
                    {i < 3 ? medalIcons[i] : <span className="text-sm text-gray-400 font-medium">{i + 1}</span>}
                  </span>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between mb-1">
                      <span className={`text-sm font-medium truncate ${i < 3 ? 'text-gray-900' : 'text-gray-700'}`}>
                        {entry.referrer_name}
                      </span>
                      <span className={`text-sm font-semibold ml-2 flex-shrink-0 ${i < 3 ? 'text-blue-700' : 'text-gray-600'}`}>
                        {entry.referral_count}
                      </span>
                    </div>
                    <div className="w-full bg-gray-100 rounded-full h-1.5">
                      <div
                        className={`h-1.5 rounded-full transition-all ${i === 0 ? 'bg-yellow-400' : i === 1 ? 'bg-gray-400' : i === 2 ? 'bg-amber-600' : 'bg-blue-300'}`}
                        style={{ width: `${(entry.referral_count / maxCount) * 100}%` }}
                      />
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* Quarterly Trends + Hired side by side */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-1">Quarterly Trends</h3>
          <p className="text-xs text-gray-400 mb-4">Referrals submitted per quarter</p>
          {quarters.length === 0 ? (
            <p className="text-gray-400 text-sm">No data yet</p>
          ) : (
            <div className="flex items-end gap-2" style={{ height: '192px' }}>
              {quarters.map((q) => (
                <div key={q.quarter} className="flex-1 flex flex-col items-center justify-end h-full">
                  <span className="text-xs font-medium text-gray-700 mb-1">{q.count}</span>
                  <div
                    className="w-full bg-blue-400 rounded-t-md transition-all"
                    style={{ height: `${Math.max((q.count / maxQuarterCount) * 160, 4)}px` }}
                  />
                  <span className="text-[10px] text-gray-400 whitespace-nowrap mt-1">
                    {q.quarter}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-1">Referral Hires by Quarter</h3>
          <p className="text-xs text-gray-400 mb-4">Candidates hired from referrals</p>
          {hiredQuarters.length === 0 ? (
            <p className="text-gray-400 text-sm">No data yet</p>
          ) : (
            <div className="flex items-end gap-2" style={{ height: '192px' }}>
              {hiredQuarters.map((q) => (
                <div key={q.quarter} className="flex-1 flex flex-col items-center justify-end h-full">
                  <span className="text-xs font-medium text-gray-700 mb-1">{q.hired}</span>
                  <div
                    className="w-full bg-emerald-400 rounded-t-md transition-all"
                    style={{ height: `${Math.max((q.hired / maxHiredCount) * 160, 4)}px` }}
                  />
                  <span className="text-[10px] text-gray-400 whitespace-nowrap mt-1">
                    {q.quarter}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Conversion rate mini-chart */}
      {allQLabels.length > 0 && (() => {
        const maxPct = Math.max(...allQLabels.map((q) => {
          const refs = referralMap[q] || 0
          const hires = hiredMap[q] || 0
          return refs > 0 ? Math.round((hires / refs) * 100) : 0
        }), 1)
        return (
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-1">Referral-to-Hire Conversion</h3>
            <p className="text-xs text-gray-400 mb-4">Percentage of referrals that resulted in a hire, by quarter</p>
            <div className="flex items-end gap-3" style={{ height: '160px' }}>
              {allQLabels.map((q) => {
                const refs = referralMap[q] || 0
                const hires = hiredMap[q] || 0
                const pct = refs > 0 ? Math.round((hires / refs) * 100) : 0
                return (
                  <div key={q} className="flex-1 flex flex-col items-center justify-end h-full">
                    <span className="text-xs font-semibold text-gray-700">{pct}%</span>
                    <span className="text-[9px] text-gray-400 mb-1">{hires}/{refs}</span>
                    <div
                      className="w-full bg-amber-400 rounded-t-md transition-all"
                      style={{ height: `${Math.max((pct / maxPct) * 120, 4)}px` }}
                    />
                    <span className="text-[10px] text-gray-400 whitespace-nowrap mt-1">{q}</span>
                  </div>
                )
              })}
            </div>
          </div>
        )
      })()}
    </div>
  )
}

function SummaryCard({ label, value, color, onClick }: { label: string; value: number; color: string; onClick?: () => void }) {
  const colorMap: Record<string, string> = {
    blue: 'bg-blue-50 text-blue-700 border-blue-200',
    green: 'bg-green-50 text-green-700 border-green-200',
    emerald: 'bg-emerald-50 text-emerald-700 border-emerald-200',
    red: 'bg-red-50 text-red-700 border-red-200',
    purple: 'bg-purple-50 text-purple-700 border-purple-200',
  }

  const base = `rounded-xl border p-5 ${colorMap[color] || colorMap.blue}`

  if (onClick) {
    return (
      <button onClick={onClick} className={`${base} text-left cursor-pointer hover:shadow-md hover:scale-[1.02] transition-all`}>
        <p className="text-sm font-medium opacity-80">{label}</p>
        <p className="text-3xl font-bold mt-1">{value}</p>
      </button>
    )
  }

  return (
    <div className={base}>
      <p className="text-sm font-medium opacity-80">{label}</p>
      <p className="text-3xl font-bold mt-1">{value}</p>
    </div>
  )
}
