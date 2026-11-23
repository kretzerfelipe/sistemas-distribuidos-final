import { useRef } from 'react'
import { ImageIcon, Trash2Icon } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/lib/format'
import type { Team } from '@/types'
import { useRemoveTeamPhoto, useUploadTeamPhoto } from './api'

const maxPhotoBytes = 2 * 1024 * 1024
const acceptedTypes = ['image/png', 'image/jpeg', 'image/webp']

export function TeamPhotoControl({ team }: { team: Team }) {
  const input = useRef<HTMLInputElement>(null)
  const upload = useUploadTeamPhoto(team.id)
  const remove = useRemoveTeamPhoto(team.id)
  const hasPhoto = team.photoUpdatedAt !== null

  const choose = (file: File | undefined) => {
    if (input.current) input.current.value = ''
    if (!file) return

    if (!acceptedTypes.includes(file.type)) {
      toast.error('Use uma imagem PNG, JPEG ou WebP.')
      return
    }
    if (file.size > maxPhotoBytes) {
      toast.error('A foto é grande demais. O limite é de 2 MB.')
      return
    }
    upload.mutate(file, {
      onSuccess: () => toast.success('Foto do time atualizada'),
      onError: (error) => toast.error(errorMessage(error)),
    })
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      <input
        ref={input}
        type="file"
        accept={acceptedTypes.join(',')}
        className="sr-only"
        aria-label="Escolher foto do time"
        onChange={(event) => choose(event.target.files?.[0])}
      />
      <Button variant="outline" size="sm" disabled={upload.isPending} onClick={() => input.current?.click()}>
        <ImageIcon />
        {hasPhoto ? 'Trocar foto' : 'Adicionar foto'}
      </Button>
      {hasPhoto ? (
        <Button
          variant="ghost"
          size="sm"
          disabled={remove.isPending}
          onClick={() =>
            remove.mutate(undefined, {
              onSuccess: () => toast.success('Foto removida'),
              onError: (error) => toast.error(errorMessage(error)),
            })
          }
        >
          <Trash2Icon />
          Remover foto
        </Button>
      ) : null}
    </div>
  )
}
