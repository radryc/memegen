import { Route, Routes } from 'react-router-dom'
import Layout from './components/Layout'
import FeedPage from './pages/FeedPage'
import CreatePage from './pages/CreatePage'
import TemplatesPage from './pages/TemplatesPage'
import MemePage from './pages/MemePage'
import LeaderboardPage from './pages/LeaderboardPage'
import SearchPage from './pages/SearchPage'
import NotFoundPage from './pages/NotFoundPage'

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<FeedPage />} />
        <Route path="create" element={<CreatePage />} />
        <Route path="templates" element={<TemplatesPage />} />
        <Route path="search" element={<SearchPage />} />
        <Route path="top" element={<LeaderboardPage />} />
        <Route path="m/:id" element={<MemePage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  )
}
