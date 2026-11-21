import type { QueryClient } from '@tanstack/react-query'
import { Outlet, createRootRouteWithContext } from '@tanstack/react-router'

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  component: () => <Outlet />,
  notFoundComponent: () => (
    <div className="grid min-h-svh place-items-center p-6 text-center">
      <div>
        <p className="font-heading text-2xl font-semibold">Página não encontrada</p>
        <a href="/" className="mt-2 inline-block text-sm text-primary underline underline-offset-4">
          Voltar ao início
        </a>
      </div>
    </div>
  ),
})
