import { RadioIcon, WifiOffIcon } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { LiveStatus } from './use-live-match'

const messages: Record<LiveStatus, string> = {
  connecting: 'Conectando ao canal ao vivo…',
  live: 'Ao vivo: os lances chegam em tempo real.',
  reconnecting: 'Conexão perdida. Reconectando…',
  finished: 'A partida terminou. O canal ao vivo foi encerrado.',
  unavailable: 'Não foi possível conectar ao canal ao vivo. A página continua atualizando a cada poucos segundos.',
}

export function LiveBanner({ status }: { status: LiveStatus }) {
  const hasProblem = status === 'reconnecting' || status === 'unavailable'
  const Icon = hasProblem ? WifiOffIcon : RadioIcon

  return (
    <p
      role="status"
      className={cn(
        'mb-4 flex items-center gap-2 rounded-lg px-3 py-2 text-sm',
        hasProblem ? 'bg-destructive/10 text-destructive' : 'bg-accent text-accent-foreground',
      )}
    >
      <Icon className="size-4 shrink-0" />
      {messages[status]}
    </p>
  )
}
