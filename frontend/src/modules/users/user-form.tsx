import { useState, type FormEvent } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { FormDialog } from '@/components/dialogs'
import { NativeSelect } from '@/components/native-select'
import { errorMessage, roleLabels } from '@/lib/format'
import type { Role } from '@/types'
import { useCreateUser, type CreateUserInput } from './api'

const minimumPasswordLength = 8
const roles: Role[] = ['USER', 'TEAM', 'ADMIN']
const emptyForm: CreateUserInput = { name: '', email: '', password: '', accountType: 'USER' }

type UserFormProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function UserForm({ open, onOpenChange }: UserFormProps) {
  const create = useCreateUser()
  const [form, setForm] = useState<CreateUserInput>(emptyForm)

  const update = <K extends keyof CreateUserInput>(key: K, value: CreateUserInput[K]) =>
    setForm((current) => ({ ...current, [key]: value }))

  const submit = (event: FormEvent) => {
    event.preventDefault()
    create.mutate(form, {
      onSuccess: () => {
        toast.success('Usuário criado')
        setForm(emptyForm)
        onOpenChange(false)
      },
    })
  }

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Novo usuário"
      description="A pessoa já pode entrar com este e-mail e esta senha."
    >
      <form onSubmit={submit} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="new-user-name">Nome</Label>
          <Input id="new-user-name" required value={form.name} onChange={(e) => update('name', e.target.value)} />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="new-user-email">E-mail</Label>
          <Input
            id="new-user-email"
            type="email"
            required
            value={form.email}
            onChange={(e) => update('email', e.target.value)}
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="new-user-password">Senha</Label>
          <Input
            id="new-user-password"
            type="password"
            autoComplete="new-password"
            minLength={minimumPasswordLength}
            required
            value={form.password}
            onChange={(e) => update('password', e.target.value)}
          />
          <p className="text-xs text-muted-foreground">Mínimo de {minimumPasswordLength} caracteres.</p>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="new-user-role">Papel</Label>
          <NativeSelect
            id="new-user-role"
            value={form.accountType}
            onChange={(e) => update('accountType', e.target.value as Role)}
          >
            {roles.map((role) => (
              <option key={role} value={role}>
                {roleLabels[role]}
              </option>
            ))}
          </NativeSelect>
        </div>
        {create.isError ? (
          <p role="alert" className="text-sm text-destructive">
            {errorMessage(create.error)}
          </p>
        ) : null}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancelar
          </Button>
          <Button type="submit" disabled={create.isPending}>
            {create.isPending ? 'Criando…' : 'Criar usuário'}
          </Button>
        </div>
      </form>
    </FormDialog>
  )
}
