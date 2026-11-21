import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { saveSession } from '@/core/auth/session'
import { errorMessage } from '@/lib/format'
import { AuthLayout } from './auth-layout'
import { useLogin } from './api'

export function LoginPage() {
  const navigate = useNavigate()
  const login = useLogin()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  const submit = (event: FormEvent) => {
    event.preventDefault()
    login.mutate(
      { email, password },
      {
        onSuccess: (result) => {
          saveSession({ token: result.accessToken, expiresAt: result.expiresAt, user: result.user })
          navigate({ to: '/' })
        },
      },
    )
  }

  return (
    <AuthLayout title="Entrar" description="Use o e-mail e a senha do seu cadastro.">
      <form onSubmit={submit} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="email">E-mail</Label>
          <Input id="email" type="email" autoComplete="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="password">Senha</Label>
          <Input
            id="password"
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>
        {login.isError ? (
          <p role="alert" className="text-sm text-destructive">
            {errorMessage(login.error)}
          </p>
        ) : null}
        <Button type="submit" className="h-10 w-full" disabled={login.isPending}>
          {login.isPending ? 'Entrando…' : 'Entrar'}
        </Button>
      </form>
      <p className="mt-6 text-sm text-muted-foreground">
        Primeira vez aqui?{' '}
        <Link to="/register" className="font-medium text-primary underline underline-offset-4">
          Criar conta
        </Link>
      </p>
    </AuthLayout>
  )
}
