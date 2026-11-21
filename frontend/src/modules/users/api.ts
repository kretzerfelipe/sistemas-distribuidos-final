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
