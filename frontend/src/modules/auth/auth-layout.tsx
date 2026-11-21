import type { ReactNode } from 'react'

export function AuthLayout({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  return (
    <div className="min-h-svh lg:grid lg:grid-cols-[1fr_1.1fr]">
      <section className="hidden flex-col justify-between bg-sidebar p-10 text-sidebar-foreground lg:flex">
        <div className="flex items-center gap-2.5">
          <span className="grid size-9 place-items-center rounded-lg bg-card-yellow font-heading text-lg font-bold text-sidebar">
            F
          </span>
          <span className="font-heading text-2xl font-semibold tracking-tight">FutHub</span>
        </div>

        <div>
          <p className="max-w-md font-heading text-4xl leading-tight font-semibold">
            Times, campeonatos e cada lance da partida em um só lugar.
          </p>
          <div className="mt-10 max-w-sm rounded-xl border border-sidebar-accent bg-sidebar-accent/60 p-5">
            <div className="flex items-center justify-between text-xs text-sidebar-muted">
              <span className="flex items-center gap-1.5">
                <span className="size-1.5 rounded-full bg-live motion-safe:animate-pulse" />
                Ao vivo
              </span>
              <span>67'</span>
            </div>
            <div className="mt-3 grid grid-cols-[1fr_auto_1fr] items-center gap-4">
              <span className="text-right font-heading text-lg">Rio Branco</span>
              <span className="font-heading text-3xl font-semibold tabular-nums">2 – 1</span>
              <span className="font-heading text-lg">Atlético</span>
            </div>
          </div>
        </div>

        <p className="text-sm text-sidebar-muted">Trabalho final de Sistemas Distribuídos, FURB.</p>
      </section>

      <section className="flex items-center justify-center px-5 py-10">
        <div className="w-full max-w-sm">
          <h1 className="text-3xl font-semibold">{title}</h1>
          <p className="mt-1.5 mb-8 text-sm text-muted-foreground">{description}</p>
          {children}
        </div>
      </section>
    </div>
  )
}
