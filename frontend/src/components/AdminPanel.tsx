export default function AdminPanel() {
  return (
    <div className="p-8 max-w-4xl mx-auto">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Admin</h1>
        <p className="text-gray-500 mt-1">This is a password-protected section. Content coming soon.</p>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6">
        <div className="flex items-center gap-3 mb-4">
          <div className="w-10 h-10 rounded-lg bg-blue-50 flex items-center justify-center">
            <span className="text-xl">🛠️</span>
          </div>
          <div>
            <h2 className="font-semibold text-gray-900">Protected Area</h2>
            <p className="text-sm text-gray-500">You have access to this section.</p>
          </div>
        </div>
        <p className="text-sm text-gray-600">
          Add any restricted content, settings, or admin tools here.
        </p>
      </div>
    </div>
  )
}
