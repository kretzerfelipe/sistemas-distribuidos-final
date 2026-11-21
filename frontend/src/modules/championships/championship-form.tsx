import { useState, type FormEvent } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { FormDialog } from '@/components/dialogs'
import { errorMessage } from '@/lib/format'
import type { Championship } from '@/types'
import { useSaveChampionship } from './api'

const defaultSlots = 10

type ChampionshipFormProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  championship?: Championship
}

export function ChampionshipForm({ open, onOpenChange, championship }: ChampionshipFormProps) {
  const save = useSaveChampionship(championship?.id)
  const [name, setName] = useState(championship?.name ?? '')
  const [slotsTotal, setSlotsTotal] = useState(championship?.slotsTotal ?? defaultSlots)
  const isEditing = championship !== undefined

  const submit = (event: FormEvent) => {
    event.preventDefault()
    save.mutate(
      { name, slotsTotal },
      {
        onSuccess: () => {
          toast.success(isEditing ? 'Campeonato atualizado' : 'Campeonato criado')
          onOpenChange(false)
          if (!isEditing) setName('')
        },
      },
    )
  }

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title={isEditing ? 'Editar campeonato' : 'Novo campeonato'}
      description="Os times se inscrevem enquanto o campeonato estiver aberto."
    >
      <form onSubmit={submit} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="championship-name">Nome</Label>
          <Input id="championship-name" required value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="championship-slots">Vagas para times</Label>
          <Input
            id="championship-slots"
            type="number"
            min={2}
            required
            value={slotsTotal}
            onChange={(e) => setSlotsTotal(Number(e.target.value))}
          />
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
            {isEditing ? 'Salvar alterações' : 'Criar campeonato'}
          </Button>
        </div>
      </form>
    </FormDialog>
  )
}
