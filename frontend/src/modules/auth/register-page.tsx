import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect } from '@/components/native-select'
import { saveSession } from '@/core/auth/session'
import { errorMessage } from '@/lib/format'
import { AuthLayout } from './auth-layout'
import { useRegister, type RegisterInput } from './api'

const minimumPasswordLength = 8

export function RegisterPage() {
  const navigate = useNavigate()
  const register = useRegister()
  const [form, setForm] = useState<RegisterInput>({ name: '', email: '', password: '', accountType: 'USER' })

  const update = <K extends keyof RegisterInput>(key: K, value: RegisterInput[K]) =>
    setForm((current) => ({ ...current, [key]: value }))

  const submit = (event: FormEvent) => {
    event.preventDefault()
    register.mutate(form, {
      onSuccess: (result) => {
        saveSession({ token: result.accessToken, expiresAt: result.expiresAt, user: result.user })
        navigate({ to: '/' })
      },
    })
  }

  return (
    <AuthLayout title="Criar sua conta" description="Escolha o tipo de conta que combina com o seu uso do FutHub.">
      <form onSubmit={submit} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="name">Nome</Label>
          <Input id="name" autoComplete="name" required value={form.name} onChange={(e) => update('name', e.target.value)} />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="email">E-mail</Label>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            required
            value={form.email}
            onChange={(e) => update('email', e.target.value)}
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="password">Senha</Label>
          <Input
            id="password"
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
          <Label htmlFor="accountType">Eu sou</Label>
          <NativeSelect
            id="accountType"
            value={form.accountType}
            onChange={(e) => update('accountType', e.target.value as RegisterInput['accountType'])}
          >
            <option value="USER">Torcedor ou jogador</option>
            <option value="TEAM">Gestor de time</option>
          </NativeSelect>
        </div>
        {register.isError ? (
          <p role="alert" className="text-sm text-destructive">
            {errorMessage(register.error)}
          </p>
        ) : null}
        <Button type="submit" className="h-10 w-full" disabled={register.isPending}>
          {register.isPending ? 'Criando conta…' : 'Criar conta'}
        </Button>
      </form>
      <p className="mt-6 text-sm text-muted-foreground">
        Já tem cadastro?{' '}
        <Link to="/login" className="font-medium text-primary underline underline-offset-4">
          Entrar
        </Link>
      </p>
    </AuthLayout>
  )
}
