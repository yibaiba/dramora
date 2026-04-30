import { useEffect, useMemo } from 'react'
import { useEpisodes, useProjects } from '../../api/hooks'
import { useStudioStore } from '../../state/studioStore'

export function useStudioSelection() {
  const {
    clearEpisode,
    selectEpisode: setEpisode,
    selectedEpisodeId,
    selectedProjectId,
    selectProject: setProject,
  } = useStudioStore()
  const { data: projects = [], isLoading: projectsLoading } = useProjects()
  const selectedProject = useMemo(
    () => projects.find((project) => project.id === selectedProjectId) ?? projects[0],
    [projects, selectedProjectId],
  )
  const { data: episodes = [], isLoading: episodesLoading } = useEpisodes(selectedProject?.id)
  const activeEpisode = useMemo(
    () => episodes.find((episode) => episode.id === selectedEpisodeId) ?? episodes[0],
    [episodes, selectedEpisodeId],
  )

  useEffect(() => {
    if (selectedProject && selectedProject.id !== selectedProjectId) {
      setProject(selectedProject.id)
    }
  }, [selectedProject, selectedProjectId, setProject])

  useEffect(() => {
    if (activeEpisode && activeEpisode.id !== selectedEpisodeId) {
      setEpisode(activeEpisode.id)
      return
    }

    if (!activeEpisode && selectedEpisodeId) {
      clearEpisode()
    }
  }, [activeEpisode, clearEpisode, selectedEpisodeId, setEpisode])

  return {
    activeEpisode,
    episodes,
    episodesLoading,
    projects,
    projectsLoading,
    selectEpisode: setEpisode,
    selectProject: (projectId: string) => {
      if (projectId === selectedProjectId) return
      setProject(projectId)
      clearEpisode()
    },
    selectedEpisodeId: activeEpisode?.id,
    selectedProject,
    selectedProjectId: selectedProject?.id,
  }
}
