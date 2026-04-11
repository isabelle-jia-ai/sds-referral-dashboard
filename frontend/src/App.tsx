import { useState, useCallback, useEffect } from 'react'
import Sidebar from './components/Sidebar'
import ReferralDashboard from './components/ReferralDashboard'
import PriorityRoles from './components/PriorityRoles'
import Analytics from './components/Analytics'
import ReferAFriend from './components/ReferAFriend'
import ProtectedTab from './components/ProtectedTab'
import AdminPanel from './components/AdminPanel'

export type Tab = 'referrals' | 'priority-roles' | 'analytics' | 'refer' | 'admin'

const validTabs: Tab[] = ['referrals', 'priority-roles', 'analytics', 'refer', 'admin']

function getInitialTab(): Tab {
  const hash = window.location.hash.replace('#', '')
  return validTabs.includes(hash as Tab) ? (hash as Tab) : 'referrals'
}

export interface Job {
  id: string
  title: string
  department: string | null
  status: string
  referral_count: number
  location: string | null
  job_url: string | null
}

function App() {
  const [activeTab, setActiveTab] = useState<Tab>(getInitialTab)
  const [initialFilter, setInitialFilter] = useState('')
  const [jobs, setJobs] = useState<Job[]>([])
  const [jobsLoading, setJobsLoading] = useState(true)

  useEffect(() => {
    const fetchJobs = async () => {
      try {
        const res = await fetch('/api/jobs')
        const data = await res.json()
        setJobs(data.jobs || [])
      } catch {
        setJobs([])
      } finally {
        setJobsLoading(false)
      }
    }
    fetchJobs()
  }, [])

  const handleTabChange = useCallback((tab: Tab) => {
    setActiveTab(tab)
    window.location.hash = tab
  }, [])

  const navigateToReferrals = useCallback((statusFilter: string) => {
    setInitialFilter(statusFilter)
    setActiveTab('referrals')
    window.location.hash = 'referrals'
  }, [])

  const clearFilter = useCallback(() => setInitialFilter(''), [])

  const renderContent = () => {
    switch (activeTab) {
      case 'referrals':
        return <ReferralDashboard initialStageFilter={initialFilter} onFilterApplied={clearFilter} />
      case 'priority-roles':
        return <PriorityRoles jobs={jobs} loading={jobsLoading} />
      case 'analytics':
        return <Analytics onStageClick={navigateToReferrals} />
      case 'refer':
        return <ReferAFriend />
      case 'admin':
        return <ProtectedTab><AdminPanel /></ProtectedTab>
      default:
        return <ReferralDashboard initialStageFilter={initialFilter} onFilterApplied={clearFilter} />
    }
  }

  return (
    <div className="flex h-screen bg-gray-50">
      <Sidebar activeTab={activeTab} setActiveTab={handleTabChange} />
      <main className="flex-1 overflow-auto">
        {renderContent()}
      </main>
    </div>
  )
}

export default App
