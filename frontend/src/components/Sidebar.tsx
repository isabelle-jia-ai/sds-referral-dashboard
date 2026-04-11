import type { Tab } from '../App'

const tabs: { id: Tab; label: string; icon: string }[] = [
  { id: 'referrals', label: 'Referrals', icon: '👥' },
  { id: 'priority-roles', label: 'Open Roles', icon: '🎯' },
  { id: 'analytics', label: 'Analytics', icon: '📊' },
  { id: 'refer', label: 'Refer a Friend', icon: '🤝' },
  { id: 'admin', label: 'Hiring Manager Analytics', icon: '🔒' },
]

export default function Sidebar({ activeTab, setActiveTab }: { activeTab: Tab; setActiveTab: (tab: Tab) => void }) {
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

      <div className="p-4 border-t border-gray-200 text-center">
        <p className="text-xs text-gray-400">Live from Datalake</p>
      </div>
    </div>
  )
}
