import {
  ChevronDown,
  Home,
  Moon,
  Plus,
  Settings,
  Sun,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { useCreateEpisode, useCreateProject } from '../../api/hooks'
import type { Project } from '../../api/types'
import { useAuthStore } from '../../state/authStore'
import { useStudioSelection } from '../hooks/useStudioSelection'
import { useThemeMode } from '../hooks/useThemeMode'
import { studioNavItems, studioRoutePaths } from '../routes'

export function StudioShell() {
  const location = useLocation()
  const {
    activeEpisode,
    episodes,
    projects,
    projectsLoading,
    selectEpisode,
    selectProject,
    selectedProject,
  } = useStudioSelection()
  const createProject = useCreateProject()
  const createEpisode = useCreateEpisode(selectedProject?.id)
  const authSession = useAuthStore((state) => state.session)
  const clearSession = useAuthStore((state) => state.clearSession)
  const { mode: themeMode, toggle: toggleTheme } = useThemeMode()
  const [projectName, setProjectName] = useState('')
  const [episodeTitle, setEpisodeTitle] = useState('')
  const activeRoute =
    useMemo(
      () =>
        studioNavItems.find(
          (item) => !item.disabled && item.path === location.pathname,
        ) ?? studioNavItems[0],
      [location.pathname],
    )

  const submitProject = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (projectName.trim() === '') return
    createProject.mutate(
      { description: 'AI 漫剧生产项目', name: projectName.trim() },
      {
        onSuccess: (project) => {
          selectProject(project.id)
          setProjectName('')
        },
      },
    )
  }

  const submitEpisode = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!selectedProject || episodeTitle.trim() === '') return
    createEpisode.mutate(
      { title: episodeTitle.trim() },
      {
        onSuccess: (episode) => {
          selectEpisode(episode.id)
          setEpisodeTitle('')
        },
      },
    )
  }

  return (
    <div className="studio-shell">
      <aside className="cinema-sidebar" aria-label="Studio 主导航">
        <div className="manmu-logo">
          <span className="logo-glyph">M</span>
          <div>
            <strong>漫幕 Manmu</strong>
            <small>AI 漫剧工场</small>
          </div>
        </div>

        <ProjectSwitcher
          isLoading={projectsLoading}
          onSelect={selectProject}
          projects={projects}
          selectedProject={selectedProject}
        />

        <nav className="cinema-nav" aria-label="Studio 页面导航">
          {studioNavItems.map((item) =>
            item.disabled ? (
              <button
                aria-disabled="true"
                className="cinema-nav-item future"
                disabled
                key={item.key}
                type="button"
              >
                <item.icon aria-hidden="true" />
                <span>{item.label}</span>
              </button>
            ) : (
              <NavLink
                className={({ isActive }) =>
                  isActive ? 'cinema-nav-item active' : 'cinema-nav-item'
                }
                end={item.path === studioRoutePaths.home}
                key={item.key}
                to={item.path}
              >
                <item.icon aria-hidden="true" />
                <span>{item.label}</span>
              </NavLink>
            ),
          )}
        </nav>

        <form className="quick-create" onSubmit={submitProject}>
          <label>
            <span>新建项目</span>
            <input
              minLength={1}
              onChange={(event) => setProjectName(event.target.value)}
              placeholder="九霄之上"
              required
              value={projectName}
            />
          </label>
          <button disabled={createProject.isPending} type="submit">
            <Plus aria-hidden="true" />
            创建项目
          </button>
        </form>

        <div className="owner-card">
          <div className="avatar-orb">
            {authSession?.user.display_name.slice(0, 2).toUpperCase() ?? 'DR'}
          </div>
          <div>
            <strong>{authSession?.user.display_name ?? 'Director'}</strong>
            <small>{authSession?.user.email ?? 'director@dramora.ai'}</small>
          </div>
          <button
            aria-label={themeMode === 'dark' ? '切换为亮色主题' : '切换为暗色主题'}
            className="owner-logout"
            onClick={toggleTheme}
            title={themeMode === 'dark' ? '切换为亮色主题' : '切换为暗色主题'}
            type="button"
          >
            {themeMode === 'dark' ? (
              <Sun aria-hidden="true" />
            ) : (
              <Moon aria-hidden="true" />
            )}
          </button>
          <button
            aria-label="退出登录"
            className="owner-logout"
            onClick={clearSession}
            title="退出登录"
            type="button"
          >
            <Settings aria-hidden="true" />
          </button>
        </div>
      </aside>

      <div className="studio-main-shell">
        <header className="shell-topbar">
          <div className="title-cluster">
            <strong>{activeRoute.label}</strong>
            <span className="pro-badge">MVP</span>
            <small>{activeRoute.description}</small>
          </div>
          <div className="shell-context-grid">
            <span className="hero-chip">
              <Home aria-hidden="true" />
              {selectedProject?.name ?? '先创建项目'}
            </span>
            <label className="episode-pill">
              <span className="sr-only">当前剧集</span>
              <select
                disabled={episodes.length === 0}
                onChange={(event) => selectEpisode(event.target.value)}
                value={activeEpisode?.id ?? ''}
              >
                {episodes.length === 0 ? <option value="">还没有剧集</option> : null}
                {episodes.map((episode) => (
                  <option key={episode.id} value={episode.id}>
                    第 {episode.number.toString().padStart(2, '0')} 集 · {episode.title}
                  </option>
                ))}
              </select>
              <ChevronDown aria-hidden="true" />
            </label>
            <form className="episode-create" onSubmit={submitEpisode}>
              <label>
                <span className="sr-only">剧集标题</span>
                <input
                  disabled={!selectedProject}
                  onChange={(event) => setEpisodeTitle(event.target.value)}
                  placeholder="新建剧集标题"
                  required
                  value={episodeTitle}
                />
              </label>
              <button disabled={!selectedProject || createEpisode.isPending} type="submit">
                <Plus aria-hidden="true" />
              </button>
            </form>
          </div>
        </header>
        <Outlet />
      </div>
    </div>
  )
}

function ProjectSwitcher({
  isLoading,
  onSelect,
  projects,
  selectedProject,
}: {
  isLoading: boolean
  onSelect: (projectId: string) => void
  projects: Project[]
  selectedProject?: Project
}) {
  return (
    <section className="project-switcher" aria-busy={isLoading} aria-label="项目切换">
      <span className="section-kicker">当前项目</span>
      <button className="project-current" disabled={projects.length === 0} type="button">
        <span className="project-cover thumb-cloud" aria-hidden="true" />
        <span>
          <strong>{selectedProject?.name ?? '新建一个漫剧项目'}</strong>
          <small>{selectedProject?.description || '九霄之上 · 天门试炼'}</small>
        </span>
        <ChevronDown aria-hidden="true" />
      </button>
      {projects.length > 1 ? (
        <div className="project-mini-list">
          {projects.slice(0, 4).map((project) => (
            <button
              className={
                project.id === selectedProject?.id ? 'mini-project active' : 'mini-project'
              }
              key={project.id}
              onClick={() => onSelect(project.id)}
              type="button"
            >
              {project.name}
            </button>
          ))}
        </div>
      ) : null}
    </section>
  )
}
