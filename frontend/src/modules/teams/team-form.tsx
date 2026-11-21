import { useState, type FormEvent } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { FormDialog } from '@/components/dialogs'
import { errorMessage } from '@/lib/format'
import type { Team } from '@/types'
import { useSaveTeam } from './api'

type TeamFormProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  team?: Team
}

export function TeamForm({ open, onOpenChange, team }: TeamFormProps) {
  const save = useSaveTeam(team?.id)
  const [name, setName] = useState(team?.name ?? '')
  const [city, setCity] = useState(team?.city ?? '')
  const isEditing = team !== undefined

  const submit = (event: FormEvent) => {
    event.preventDefault()
    save.mutate(
      { name, city },
      {
        onSuccess: () => {
          toast.success(isEditing ? 'Time atualizado' : 'Time criado')
          onOpenChange(false)
          if (!isEditing) {
            setName('')
            setCity('')
          }
        },
      },
    )
  }

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
