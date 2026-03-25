import { useState, useEffect } from 'react'
import type { Tab } from '../App'

interface SyncStatus {
  last_sync: {
    sync_type: string
    status: string
    records_synced: number
    started_at: string
    finished_at: string | null
  } | null
}

const tabs: { id: Tab; label: string; icon: string }[] = [
  { id: 'referrals', label: 'Referrals', icon: '👥' },
  { id: 'priority-roles', label: 'Priority Roles', icon: '🎯' },
  { id: 'analytics', label: 'Analytics', icon: '📊' },
]

export default function Sidebar({ activeTab, setActiveTab }: { activeTab: Tab; setActiveTab: (tab: Tab) => void }) {
  const [syncStatus, setSyncStatus] = useState<SyncStatus | null>(null)

  useEffect(() => {
    const fetchSync = async () => {
      try {
        const res = await fetch('/api/sync/status')
        const data = await res.json()
        setSyncStatus(data)
      } catch { /* ignore */ }
    }
    fetchSync()
    const interval = setInterval(fetchSync, 60000)
    return () => clearInterval(interval)
  }, [])

  const handleSync = async () => {
    try {
      await fetch('/api/sync/trigger', { method: 'POST' })
      setTimeout(async () => {
        const res = await fetch('/api/sync/status')
        setSyncStatus(await res.json())
      }, 3000)
    } catch { /* ignore */ }
  }

  const formatTime = (ts: string) => {
    const d = new Date(ts)
    return d.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  }

  return (
    <div className="w-64 bg-white border-r border-gray-200 flex flex-col">
      <div className="p-6 border-b border-gray-200">
        <h1 className="text-xl font-bold text-gray-900">SDS Referrals</h1>
        <p className="text-sm text-gray-500 mt-1">Dashboard</p>
      </div>

      <nav className="flex-1 p-4 space-y-1">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
              activeTab === tab.id
                ? 'bg-blue-50 text-blue-700'
                : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
            }`}
          >
            <span className="text-lg">{tab.icon}</span>
            {tab.label}
          </button>
        ))}
      </nav>

      <div className="p-4 border-t border-gray-200">
        <button
          onClick={handleSync}
          className="w-full px-3 py-2 text-sm font-medium text-blue-600 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors"
        >
          Sync from Greenhouse
        </button>
        {syncStatus?.last_sync && (
          <p className="text-xs text-gray-400 mt-2 text-center">
            Last sync: {formatTime(syncStatus.last_sync.started_at)}
            {syncStatus.last_sync.status === 'success' && (
              <span className="text-green-500 ml-1">✓</span>
            )}
            {syncStatus.last_sync.status === 'error' && (
              <span className="text-red-500 ml-1">✗</span>
            )}
          </p>
        )}
      </div>
    </div>
  )
}
