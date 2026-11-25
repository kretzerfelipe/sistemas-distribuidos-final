import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { PlusIcon } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { NativeSelect } from '@/components/native-select'
import { ErrorState, LoadingRows } from '@/components/state'
import { PageHeader } from '@/components/page-header'
import { useSession } from '@/core/auth/session'
import { errorMessage, formatDate, roleLabels } from '@/lib/format'
import type { Role } from '@/types'
import { useUpdateRole, usersQuery } from './api'
import { UserForm } from './user-form'

const roles: Role[] = ['USER', 'TEAM', 'ADMIN']

export function UsersPage() {
  const session = useSession()
  const users = useQuery(usersQuery)
  const updateRole = useUpdateRole()
  const [formIsOpen, setFormIsOpen] = useState(false)

  const changeRole = (id: string, role: Role) =>
    updateRole.mutate(
      { id, role },
      {
        onSuccess: () => toast.success('Papel atualizado. Vale a partir do próximo login da pessoa.'),
        onError: (error) => toast.error(errorMessage(error)),
      },
    )

  return (
    <>
      <PageHeader
        title="Usuários"
        description="O novo papel vale a partir do próximo login da pessoa."
        actions={
          <Button onClick={() => setFormIsOpen(true)}>
            <PlusIcon />
            Novo usuário
          </Button>
        }
      />

      {users.isPending ? <LoadingRows /> : null}
      {users.isError ? <ErrorState error={users.error} onRetry={() => users.refetch()} /> : null}
      {users.isSuccess ? (
        <ul className="divide-y rounded-xl border bg-card">
          {users.data.map((user) => (
            <li key={user.id} className="grid items-center gap-3 px-4 py-3 sm:grid-cols-[1fr_11rem]">
              <div className="min-w-0">
                <p className="truncate font-medium">{user.name}</p>
                <p className="truncate text-sm text-muted-foreground">
                  {user.email} · cadastrado em {formatDate(user.createdAt)}
                </p>
              </div>
              <NativeSelect
                aria-label={`Papel de ${user.name}`}
                value={user.role}
                disabled={updateRole.isPending || user.id === session?.user.id}
                onChange={(e) => changeRole(user.id, e.target.value as Role)}
              >
                {roles.map((role) => (
                  <option key={role} value={role}>
                    {roleLabels[role]}
                  </option>
                ))}
              </NativeSelect>
            </li>
          ))}
        </ul>
      ) : null}

      <UserForm open={formIsOpen} onOpenChange={setFormIsOpen} />
    </>
  )
}
