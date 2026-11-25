import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { ChevronRightIcon, PlusIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { EmptyState, ErrorState, LoadingRows } from '@/components/state'
import { PageHeader } from '@/components/page-header'
import { useSession } from '@/core/auth/session'
import type { Team } from '@/types'
import { myTeamsQuery, teamsQuery } from './api'
import { TeamAvatar } from './team-avatar'
import { TeamForm } from './team-form'

function TeamRow({ team }: { team: Team }) {
  return (
    <li>
      <Link
        to="/teams/$teamId"
        params={{ teamId: team.id }}
        className="flex items-center justify-between gap-3 px-4 py-3.5 transition-colors hover:bg-secondary/60 focus-visible:bg-secondary/60 focus-visible:outline-none"
      >
        <div className="flex min-w-0 items-center gap-3">
          <TeamAvatar team={team} />
          <div className="min-w-0">
            <p className="truncate font-medium">{team.name}</p>
            <p className="truncate text-sm text-muted-foreground">{team.city}</p>
          </div>
        </div>
        <ChevronRightIcon className="size-4 shrink-0 text-muted-foreground" />
      </Link>
    </li>
  )
}

export function TeamsPage() {
  const session = useSession()
  const isManager = session?.user.role === 'TEAM'
  const canCreate = isManager || session?.user.role === 'ADMIN'
  const [scope, setScope] = useState<'all' | 'mine'>('all')
  const [formIsOpen, setFormIsOpen] = useState(false)
  const allTeams = useQuery(teamsQuery)
  const myTeams = useQuery({ ...myTeamsQuery, enabled: isManager })
  const list = scope === 'mine' ? myTeams : allTeams

  return (
    <>
      <PageHeader
        title="Times"
        description="Todos os times cadastrados. Abra um para ver o elenco."
        actions={
          canCreate ? (
            <Button onClick={() => setFormIsOpen(true)}>
              <PlusIcon />
              Novo time
            </Button>
          ) : null
        }
      />

      {isManager ? (
        <Tabs value={scope} onValueChange={(value) => setScope(value as 'all' | 'mine')} className="mb-4">
          <TabsList>
            <TabsTrigger value="all">Todos os times</TabsTrigger>
            <TabsTrigger value="mine">Meus times</TabsTrigger>
          </TabsList>
        </Tabs>
      ) : null}

      {list.isPending ? <LoadingRows /> : null}
      {list.isError ? <ErrorState error={list.error} onRetry={() => list.refetch()} /> : null}
      {list.isSuccess && list.data.length === 0 ? (
        <EmptyState
          title={scope === 'mine' ? 'Você ainda não tem times' : 'Nenhum time cadastrado ainda'}
          description={canCreate ? 'Crie um time para começar a montar o elenco.' : undefined}
          action={canCreate ? <Button onClick={() => setFormIsOpen(true)}>Novo time</Button> : undefined}
        />
      ) : null}
      {list.isSuccess && list.data.length > 0 ? (
        <ul className="divide-y overflow-hidden rounded-xl border bg-card">
          {list.data.map((team) => (
            <TeamRow key={team.id} team={team} />
          ))}
        </ul>
      ) : null}

      {canCreate ? <TeamForm open={formIsOpen} onOpenChange={setFormIsOpen} /> : null}
    </>
  )
}
