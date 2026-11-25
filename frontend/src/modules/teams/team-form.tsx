import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import { ImageIcon, XIcon } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { FormDialog } from '@/components/dialogs'
import { errorMessage } from '@/lib/format'
import type { Team } from '@/types'
import { useSaveTeam } from './api'
import { acceptedPhotoTypes, photoFileProblem } from './photo-file'
import { TeamAvatar } from './team-avatar'

type TeamFormProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  team?: Team
}

export function TeamForm({ open, onOpenChange, team }: TeamFormProps) {
  const save = useSaveTeam(team?.id)
  const photoInput = useRef<HTMLInputElement>(null)
  const [name, setName] = useState(team?.name ?? '')
  const [city, setCity] = useState(team?.city ?? '')
  const [photo, setPhoto] = useState<File | null>(null)
  const [photoError, setPhotoError] = useState<string | null>(null)
  const isEditing = team !== undefined
  const photoPreview = useMemo(() => (photo ? URL.createObjectURL(photo) : null), [photo])

  useEffect(() => {
    if (!photoPreview) return
    return () => URL.revokeObjectURL(photoPreview)
  }, [photoPreview])

  const choosePhoto = (file: File | undefined) => {
    if (photoInput.current) photoInput.current.value = ''
    if (!file) return
    const problem = photoFileProblem(file)
    setPhotoError(problem)
    if (!problem) setPhoto(file)
  }

  const clearPhoto = () => {
    setPhoto(null)
    setPhotoError(null)
  }

  const submit = (event: FormEvent) => {
    event.preventDefault()
    save.mutate(
      { name, city, photo },
      {
        onSuccess: ({ photoFailed }) => {
          toast.success(isEditing ? 'Time atualizado' : 'Time criado')
          if (photoFailed) toast.error('A foto não foi enviada. Tente de novo pela página do time.')
          onOpenChange(false)
          clearPhoto()
          if (!isEditing) {
            setName('')
            setCity('')
          }
        },
      },
    )
  }

  const hasPhotoToShow = photoPreview !== null || (team?.photoUpdatedAt ?? null) !== null

  return (
    <FormDialog open={open} onOpenChange={onOpenChange} title={isEditing ? 'Editar time' : 'Novo time'}>
      <form onSubmit={submit} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="team-name">Nome do time</Label>
          <Input id="team-name" required value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="team-city">Cidade</Label>
          <Input id="team-city" required value={city} onChange={(e) => setCity(e.target.value)} />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="team-photo">Escudo ou foto do time (opcional)</Label>
          <div className="flex items-center gap-3">
            {photoPreview ? (
              <img src={photoPreview} alt="Prévia da foto escolhida" className="size-12 rounded-lg object-cover" />
            ) : isEditing && hasPhotoToShow ? (
              <TeamAvatar team={team} className="size-12" />
            ) : (
              <span className="grid size-12 place-items-center rounded-lg bg-secondary text-muted-foreground">
                <ImageIcon className="size-5" />
              </span>
            )}
            <input
              ref={photoInput}
              id="team-photo"
              type="file"
              accept={acceptedPhotoTypes.join(',')}
              className="sr-only"
              onChange={(event) => choosePhoto(event.target.files?.[0])}
            />
            <Button type="button" variant="outline" size="sm" onClick={() => photoInput.current?.click()}>
              {hasPhotoToShow ? 'Trocar imagem' : 'Escolher imagem'}
            </Button>
            {photo ? (
              <Button type="button" variant="ghost" size="sm" onClick={clearPhoto}>
                <XIcon />
                Descartar
              </Button>
            ) : null}
          </div>
          <p className="text-xs text-muted-foreground">PNG, JPEG ou WebP, até 2 MB.</p>
          {photoError ? (
            <p role="alert" className="text-sm text-destructive">
              {photoError}
            </p>
          ) : null}
        </div>
        {save.isError ? (
          <p role="alert" className="text-sm text-destructive">
            {errorMessage(save.error)}
          </p>
        ) : null}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancelar
          </Button>
          <Button type="submit" disabled={save.isPending}>
            {isEditing ? 'Salvar alterações' : 'Criar time'}
          </Button>
        </div>
      </form>
    </FormDialog>
  )
}
