import { useRef } from 'react'
import { ImageIcon, Trash2Icon } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/lib/format'
import type { Team } from '@/types'
import { useRemoveTeamPhoto, useUploadTeamPhoto } from './api'
import { acceptedPhotoTypes, photoFileProblem } from './photo-file'


export function TeamPhotoControl({ team }: { team: Team }) {
  const input = useRef<HTMLInputElement>(null)
  const upload = useUploadTeamPhoto(team.id)
  const remove = useRemoveTeamPhoto(team.id)
  const hasPhoto = team.photoUpdatedAt !== null

  const choose = (file: File | undefined) => {
    if (input.current) input.current.value = ''
    if (!file) return

    const problem = photoFileProblem(file)
    if (problem) {
      toast.error(problem)
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
        accept={acceptedPhotoTypes.join(',')}
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
