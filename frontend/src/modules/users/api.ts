import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { Role, User } from '@/types'

export const usersQuery = queryOptions({
  queryKey: ['users'],
  queryFn: () => api<User[]>('/users'),
})

export function useUpdateRole() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, role }: { id: string; role: Role }) =>
      api<User>(`/users/${id}/role`, { method: 'PATCH', body: { role } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: usersQuery.queryKey }),
  })
}

export type CreateUserInput = {
  name: string
  email: string
  password: string
  accountType: Role
}

export function useCreateUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateUserInput) => api<User>('/auth/users', { method: 'POST', body: input }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: usersQuery.queryKey }),
  })
}
