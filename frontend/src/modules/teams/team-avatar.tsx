import { useState } from 'react'
import { mainBaseUrl } from '@/lib/api'
import { cn } from '@/lib/utils'

type TeamAvatarProps = {
  team: { id: string; name: string; photoUpdatedAt: string | null }
  className?: string
}

export function teamPhotoUrl(team: { id: string; photoUpdatedAt: string | null }) {
  return `${mainBaseUrl}/teams/${team.id}/photo?v=${encodeURIComponent(team.photoUpdatedAt ?? '')}`
}

export function TeamAvatar({ team, className }: TeamAvatarProps) {
  const [failedUrl, setFailedUrl] = useState<string | null>(null)
  const url = team.photoUpdatedAt ? teamPhotoUrl(team) : null
  const showsPhoto = url !== null && failedUrl !== url
  const initials = team.name.trim().slice(0, 2).toUpperCase()

  return (
    <span
      className={cn(
        'grid size-10 shrink-0 place-items-center overflow-hidden rounded-lg bg-accent font-heading text-sm font-semibold text-accent-foreground',
        className,
      )}
    >
      {showsPhoto ? (
        <img src={url} alt={`Escudo de ${team.name}`} className="size-full object-cover" onError={() => setFailedUrl(url)} />
      ) : (
        <span aria-hidden>{initials}</span>
      )}
    </span>
  )
}
