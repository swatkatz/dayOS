import { Outlet } from 'react-router-dom'
import Sidebar from './Sidebar'

export default function Layout() {
  return (
    <div className="flex flex-col md:flex-row min-h-screen bg-bg-primary">
      <Sidebar />
      <main className="flex-1 px-3 py-4 md:p-6 pb-20 md:pb-6">
        <Outlet />
      </main>
    </div>
  )
}
