import { useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { LoginResult, User } from '@/types'

export type LoginInput = { email: string; password: string }

export type RegisterInput = {
  name: string
  email: string
  password: string
  accountType: 'USER' | 'TEAM'
}

export function useLogin() {
  return useMutation({
    mutationFn: (input: LoginInput) => api<LoginResult>('/auth/login', { method: 'POST', body: input }),
  })
}

export function useRegister() {
  return useMutation({
    mutationFn: async (input: RegisterInput) => {
      await api<User>('/auth/register', { method: 'POST', body: input })
      return api<LoginResult>('/auth/login', {
        method: 'POST',
        body: { email: input.email, password: input.password },
      })
    },
  })
}
