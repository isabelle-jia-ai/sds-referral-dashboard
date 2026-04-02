import { useState, useEffect, useCallback } from 'react'

interface Job {
  id: string
  title: string
}

interface FormData {
  referrer_email: string
  candidate_name: string
  candidate_email: string
  linkedin_url: string
  phone: string
  job_id: string
  job_title: string
  relationship: string
  note: string
}

const STEPS = ['Your Info', 'Candidate Info', 'Select Role', 'Additional Context', 'Review & Submit']

const emptyForm: FormData = {
  referrer_email: '',
  candidate_name: '',
  candidate_email: '',
  linkedin_url: '',
  phone: '',
  job_id: '',
  job_title: '',
  relationship: '',
  note: '',
}

export default function ReferAFriend() {
  const [step, setStep] = useState(0)
  const [form, setForm] = useState<FormData>(emptyForm)
  const [jobs, setJobs] = useState<Job[]>([])
  const [jobSearch, setJobSearch] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [result, setResult] = useState<'success' | 'error' | null>(null)
  const [errorMsg, setErrorMsg] = useState('')

  useEffect(() => {
    fetch('/api/refer/jobs')
      .then((r) => r.json())
      .then((d) => setJobs(d.jobs || []))
      .catch(() => {})
  }, [])

  const set = useCallback(
    (field: keyof FormData, value: string) => setForm((prev) => ({ ...prev, [field]: value })),
    [],
  )

  const canProceed = (): boolean => {
    switch (step) {
      case 0:
        return form.referrer_email.includes('@')
      case 1:
        return form.candidate_name.trim() !== '' && form.candidate_email.includes('@')
      case 2:
        return form.job_id !== ''
      case 3:
        return true
      default:
        return true
    }
  }

  const handleSubmit = async () => {
    setSubmitting(true)
    setResult(null)
    setErrorMsg('')
    try {
      const res = await fetch('/api/refer', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || `Request failed (${res.status})`)
      }
      setResult('success')
    } catch (e) {
      setResult('error')
      setErrorMsg(e instanceof Error ? e.message : 'Submission failed')
    } finally {
      setSubmitting(false)
    }
  }

  const reset = () => {
    setForm(emptyForm)
    setStep(0)
    setResult(null)
    setErrorMsg('')
  }

  if (result === 'success') {
    return (
      <div className="p-6 max-w-2xl mx-auto">
        <div className="bg-white rounded-xl border border-green-200 p-10 text-center">
          <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h2 className="text-2xl font-bold text-gray-900 mb-2">Referral Submitted!</h2>
          <p className="text-gray-500 mb-6">
            <strong>{form.candidate_name}</strong> has been referred for <strong>{form.job_title}</strong>.
            They'll be added to the Application Review stage automatically.
          </p>
          <button
            onClick={reset}
            className="px-6 py-2.5 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition-colors"
          >
            Refer Someone Else
          </button>
        </div>
      </div>
    )
  }

  const filteredJobs = jobs.filter((j) =>
    jobSearch === '' || j.title.toLowerCase().includes(jobSearch.toLowerCase()),
  )

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <div className="mb-8">
        <h2 className="text-2xl font-bold text-gray-900">Refer a Friend</h2>
        <p className="text-sm text-gray-500 mt-1">Submit a referral directly to the recruiting pipeline</p>
      </div>

      {/* Progress bar */}
      <div className="mb-8">
        <div className="flex items-center justify-between mb-2">
          {STEPS.map((label, i) => (
            <button
              key={label}
              onClick={() => i < step && setStep(i)}
              disabled={i > step}
              className={`text-xs font-medium transition-colors ${
                i === step
                  ? 'text-blue-600'
                  : i < step
                  ? 'text-blue-400 cursor-pointer hover:text-blue-600'
                  : 'text-gray-300'
              }`}
            >
              {label}
            </button>
          ))}
        </div>
        <div className="w-full bg-gray-100 rounded-full h-1.5">
          <div
            className="bg-blue-500 h-1.5 rounded-full transition-all duration-300"
            style={{ width: `${((step + 1) / STEPS.length) * 100}%` }}
          />
        </div>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 p-6">
        {/* Step 0: Your Info */}
        {step === 0 && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-gray-900">Your Information</h3>
            <p className="text-sm text-gray-500">We'll use your email to credit the referral to you in Greenhouse.</p>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Your Work Email *</label>
              <input
                type="email"
                value={form.referrer_email}
                onChange={(e) => set('referrer_email', e.target.value)}
                placeholder="you@applied.co"
                className="w-full px-4 py-2.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>
        )}

        {/* Step 1: Candidate Info */}
        {step === 1 && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-gray-900">Candidate Information</h3>
            <p className="text-sm text-gray-500">Tell us about the person you're referring.</p>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Full Name *</label>
              <input
                type="text"
                value={form.candidate_name}
                onChange={(e) => set('candidate_name', e.target.value)}
                placeholder="Jane Smith"
                className="w-full px-4 py-2.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Email *</label>
              <input
                type="email"
                value={form.candidate_email}
                onChange={(e) => set('candidate_email', e.target.value)}
                placeholder="jane@example.com"
                className="w-full px-4 py-2.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">LinkedIn URL</label>
              <input
                type="url"
                value={form.linkedin_url}
                onChange={(e) => set('linkedin_url', e.target.value)}
                placeholder="https://linkedin.com/in/..."
                className="w-full px-4 py-2.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Phone</label>
              <input
                type="tel"
                value={form.phone}
                onChange={(e) => set('phone', e.target.value)}
                placeholder="+1 555-123-4567"
                className="w-full px-4 py-2.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>
        )}

        {/* Step 2: Role Selection */}
        {step === 2 && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-gray-900">Select a Role</h3>
            <p className="text-sm text-gray-500">Choose the open position this person is best suited for.</p>
            <input
              type="text"
              value={jobSearch}
              onChange={(e) => setJobSearch(e.target.value)}
              placeholder="Search roles..."
              className="w-full px-4 py-2.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
            <div className="max-h-64 overflow-y-auto space-y-1 border border-gray-100 rounded-lg p-1">
              {filteredJobs.length === 0 ? (
                <p className="text-sm text-gray-400 text-center py-4">No matching roles</p>
              ) : (
                filteredJobs.map((job) => (
                  <button
                    key={job.id}
                    onClick={() => {
                      set('job_id', job.id)
                      set('job_title', job.title)
                    }}
                    className={`w-full text-left px-3 py-2.5 rounded-lg text-sm transition-colors ${
                      form.job_id === job.id
                        ? 'bg-blue-50 text-blue-700 font-medium'
                        : 'text-gray-700 hover:bg-gray-50'
                    }`}
                  >
                    {job.title}
                    {form.job_id === job.id && (
                      <svg className="w-4 h-4 inline ml-2 text-blue-500" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                      </svg>
                    )}
                  </button>
                ))
              )}
            </div>
          </div>
        )}

        {/* Step 3: Additional Context */}
        {step === 3 && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-gray-900">Additional Context</h3>
            <p className="text-sm text-gray-500">Help the recruiting team understand your referral better.</p>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">How do you know this person?</label>
              <select
                value={form.relationship}
                onChange={(e) => set('relationship', e.target.value)}
                className="w-full px-4 py-2.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="">Select...</option>
                <option value="Former colleague">Former colleague</option>
                <option value="Current colleague">Current colleague</option>
                <option value="Friend">Friend</option>
                <option value="Classmate">Classmate / Alumni</option>
                <option value="Professional network">Professional network</option>
                <option value="Other">Other</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Why are you recommending them?</label>
              <textarea
                value={form.note}
                onChange={(e) => set('note', e.target.value)}
                rows={4}
                placeholder="What makes this person a great fit for this role?"
                className="w-full px-4 py-2.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
              />
            </div>
          </div>
        )}

        {/* Step 4: Review */}
        {step === 4 && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-gray-900">Review Your Referral</h3>
            <p className="text-sm text-gray-500">Please confirm the details below before submitting.</p>

            <div className="bg-gray-50 rounded-lg p-4 space-y-3">
              <ReviewRow label="Your Email" value={form.referrer_email} />
              <ReviewRow label="Candidate" value={form.candidate_name} />
              <ReviewRow label="Candidate Email" value={form.candidate_email} />
              {form.linkedin_url && <ReviewRow label="LinkedIn" value={form.linkedin_url} />}
              {form.phone && <ReviewRow label="Phone" value={form.phone} />}
              <ReviewRow label="Role" value={form.job_title} />
              {form.relationship && <ReviewRow label="Relationship" value={form.relationship} />}
              {form.note && <ReviewRow label="Note" value={form.note} />}
            </div>

            {result === 'error' && (
              <div className="bg-red-50 border border-red-200 rounded-lg p-3">
                <p className="text-sm text-red-700">{errorMsg}</p>
              </div>
            )}
          </div>
        )}

        {/* Navigation */}
        <div className="flex items-center justify-between mt-6 pt-4 border-t border-gray-100">
          {step > 0 ? (
            <button
              onClick={() => setStep((s) => s - 1)}
              className="px-4 py-2 text-sm font-medium text-gray-600 hover:text-gray-900 transition-colors"
            >
              Back
            </button>
          ) : (
            <div />
          )}

          {step < STEPS.length - 1 ? (
            <button
              onClick={() => setStep((s) => s + 1)}
              disabled={!canProceed()}
              className="px-6 py-2.5 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            >
              Continue
            </button>
          ) : (
            <button
              onClick={handleSubmit}
              disabled={submitting}
              className="px-6 py-2.5 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 disabled:opacity-60 transition-colors flex items-center gap-2"
            >
              {submitting && (
                <div className="animate-spin rounded-full h-4 w-4 border-2 border-white border-t-transparent" />
              )}
              {submitting ? 'Submitting...' : 'Submit Referral'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

function ReviewRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start gap-3">
      <span className="text-xs font-medium text-gray-500 w-28 flex-shrink-0 pt-0.5">{label}</span>
      <span className="text-sm text-gray-900 break-all">{value}</span>
    </div>
  )
}
