import { createFileRoute } from '@tanstack/react-router'
import { redirectIfSignedIn } from '@/core/auth/guards'
import { LoginPage } from '@/modules/auth/login-page'

export const Route = createFileRoute('/login')({
  beforeLoad: redirectIfSignedIn,
  component: LoginPage,
})
