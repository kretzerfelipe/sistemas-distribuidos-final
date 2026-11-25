import { useState, type ComponentType } from 'react'
import { Link, Outlet, useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { BellIcon, CalendarDaysIcon, LogOutIcon, SparklesIcon, MailIcon, MenuIcon, ShieldIcon, ShirtIcon, TrophyIcon, UserRoundIcon, UsersIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Sheet, SheetContent, SheetDescription, SheetTitle } from '@/components/ui/sheet'
import { clearSession, useSession } from '@/core/auth/session'
import { roleLabels } from '@/lib/format'
import type { Role } from '@/types'

type NavItem = {
  to: '/games' | '/championships' | '/premium' | '/notifications' | '/teams' | '/player' | '/invitations' | '/users'
  label: string
  icon: ComponentType<{ className?: string }>
  roles?: Role[]
}

const navItems: NavItem[] = [
  { to: '/games', label: 'Jogos', icon: CalendarDaysIcon },
  { to: '/championships', label: 'Campeonatos', icon: TrophyIcon },
  { to: '/teams', label: 'Times', icon: UsersIcon },
  { to: '/premium', label: 'Premium', icon: SparklesIcon, roles: ['USER', 'TEAM'] },
  { to: '/notifications', label: 'Notificações', icon: BellIcon, roles: ['USER', 'TEAM'] },
  { to: '/player', label: 'Meu jogador', icon: ShirtIcon, roles: ['USER'] },
  { to: '/invitations', label: 'Convites', icon: MailIcon, roles: ['USER'] },
  { to: '/users', label: 'Usuários', icon: ShieldIcon, roles: ['ADMIN'] },
]

function Brand() {
  return (
    <Link to="/games" className="flex items-center gap-2.5 px-2">
      <span className="grid size-8 place-items-center rounded-lg bg-card-yellow font-heading text-base font-bold text-sidebar">
        F
      </span>
      <span className="font-heading text-xl font-semibold tracking-tight text-sidebar-foreground">FutHub</span>
    </Link>
  )
}

function NavLinks({ role, onNavigate }: { role: Role; onNavigate?: () => void }) {
  const visibleItems = navItems.filter((item) => !item.roles || item.roles.includes(role))
  return (
    <nav className="flex flex-col gap-1" aria-label="Principal">
      {visibleItems.map((item) => (
        <Link
          key={item.to}
          to={item.to}
          onClick={onNavigate}
          className="flex items-center gap-3 rounded-lg px-3 py-2 text-sm text-sidebar-muted transition-colors hover:bg-sidebar-accent hover:text-sidebar-foreground"
          activeProps={{ className: 'bg-sidebar-accent !text-sidebar-foreground font-medium' }}
        >
          <item.icon className="size-4" />
          {item.label}
        </Link>
      ))}
    </nav>
  )
}

function Account({ onNavigate }: { onNavigate?: () => void }) {
  const session = useSession()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  if (!session) return null

  const signOut = () => {
    onNavigate?.()
    clearSession()
    queryClient.clear()
    navigate({ to: '/login' })
  }

  return (
    <div className="border-t border-sidebar-accent pt-4">
      <div className="flex items-center gap-3 px-2">
        <span className="grid size-9 shrink-0 place-items-center rounded-full bg-sidebar-accent text-sidebar-foreground">
          <UserRoundIcon className="size-4" />
        </span>
        <div className="min-w-0">
          <p className="truncate text-sm font-medium text-sidebar-foreground">{session.user.name}</p>
          <p className="truncate text-xs text-sidebar-muted">{roleLabels[session.user.role]}</p>
        </div>
      </div>
      <Button
        variant="ghost"
        className="mt-3 w-full justify-start text-sidebar-muted hover:bg-sidebar-accent hover:text-sidebar-foreground"
        onClick={signOut}
      >
        <LogOutIcon />
        Sair
      </Button>
    </div>
  )
}

export function AppShell() {
  const session = useSession()
  const [menuIsOpen, setMenuIsOpen] = useState(false)
  if (!session) return null
  const role = session.user.role

  return (
    <div className="min-h-svh lg:grid lg:grid-cols-[15rem_1fr]">
      <aside className="sticky top-0 hidden h-svh flex-col justify-between bg-sidebar p-4 lg:flex">
        <div className="space-y-8 pt-2">
          <Brand />
          <NavLinks role={role} />
        </div>
        <Account />
      </aside>

      <header className="sticky top-0 z-20 flex items-center justify-between bg-sidebar px-4 py-3 lg:hidden">
        <Brand />
        <Button
          variant="ghost"
          size="icon"
          aria-label="Abrir menu"
          className="text-sidebar-foreground hover:bg-sidebar-accent"
          onClick={() => setMenuIsOpen(true)}
        >
          <MenuIcon />
        </Button>
      </header>

      <Sheet open={menuIsOpen} onOpenChange={setMenuIsOpen}>
        <SheetContent side="left" className="flex w-64 flex-col justify-between border-0 bg-sidebar p-4">
          <SheetTitle className="sr-only">Menu</SheetTitle>
          <SheetDescription className="sr-only">Navegue entre as seções</SheetDescription>
          <div className="space-y-8 pt-2">
            <Brand />
            <NavLinks role={role} onNavigate={() => setMenuIsOpen(false)} />
          </div>
          <Account onNavigate={() => setMenuIsOpen(false)} />
        </SheetContent>
      </Sheet>

      <main className="mx-auto w-full max-w-5xl px-4 py-6 sm:px-6 lg:px-10 lg:py-10">
        <Outlet />
      </main>
    </div>
  )
}
