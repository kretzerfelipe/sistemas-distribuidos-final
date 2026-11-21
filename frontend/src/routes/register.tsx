import { createFileRoute } from '@tanstack/react-router'
import { redirectIfSignedIn } from '@/core/auth/guards'
import { RegisterPage } from '@/modules/auth/register-page'

export const Route = createFileRoute('/register')({
  beforeLoad: redirectIfSignedIn,
  component: RegisterPage,
})
