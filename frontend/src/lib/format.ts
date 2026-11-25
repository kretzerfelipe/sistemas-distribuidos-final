import { ApiError } from '@/lib/api'
import type {
  ChampionshipStatus,
  InvitationStatus,
  MatchEventType,
  MatchStatus,
  Position,
  Role,
} from '@/types'

const dateTimeFormat = new Intl.DateTimeFormat('pt-BR', { dateStyle: 'medium', timeStyle: 'short' })
const dateFormat = new Intl.DateTimeFormat('pt-BR', { dateStyle: 'medium' })

export function formatDateTime(value: string) {
  return dateTimeFormat.format(new Date(value))
}

export function formatDate(value: string) {
  return dateFormat.format(new Date(value))
}

export function toDateTimeLocalValue(value: string) {
  const date = new Date(value)
  const offsetMs = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16)
}

const genericErrorMessage = 'Algo deu errado. Tente novamente.'

const errorMessagesByCode: Record<string, string> = {
  network_error: 'Não foi possível acessar o servidor. Confira se o serviço main está no ar.',
  invalid_request: 'A requisição está incompleta ou inválida.',
  unauthorized: 'Sua sessão expirou. Entre novamente.',
  invalid_credentials: 'E-mail ou senha incorretos.',
  forbidden: 'Você não tem permissão para fazer isso.',
  not_found: 'Não encontramos o que você procura.',
  player_not_in_team: 'Esse jogador não faz parte do time.',
  invalid_input: 'Os dados informados não são válidos.',
  invalid_state: 'Essa ação não é permitida no estado atual.',
  email_taken: 'Já existe uma conta com esse e-mail.',
  player_already_exists: 'Você já tem um perfil de jogador.',
  team_name_taken: 'Já existe um time com esse nome.',
  team_in_use: 'Esse time está inscrito em um campeonato e não pode ser excluído.',
  player_already_in_team: 'Esse jogador já faz parte de um time.',
  invitation_already_pending: 'Esse jogador já tem um convite pendente do seu time.',
  invitation_not_pending: 'Esse convite não está mais pendente.',
  championship_full: 'O campeonato não tem mais vagas.',
  championship_not_open: 'O campeonato não está mais aberto para inscrições.',
  championship_has_matches: 'O campeonato tem partidas e não pode ser excluído.',
  team_already_enrolled: 'Esse time já está inscrito no campeonato.',
  team_not_enrolled: 'Esse time não está inscrito no campeonato.',
  unsupported_photo: 'Use uma imagem PNG, JPEG ou WebP.',
  photo_too_large: 'A foto é grande demais. O limite é de 2 MB.',
  storage_unavailable: 'Não foi possível salvar a foto agora. Tente novamente em instantes.',
  subscription_in_force:'Você já tem uma assinatura em andamento ou ativa.',
  match_not_live:'Essa partida não está ao vivo.',
  premium_required: 'O acompanhamento em tempo real é exclusivo para assinantes.',
  premium_unavailable: 'Não foi possível verificar a assinatura agora. Tente novamente.',
  cache_unavailable: 'Os dados ao vivo estão indisponíveis no momento. Tente novamente em instantes.',
  internal_error:'O servidor encontrou um erro. Tente novamente em instantes.',
}

const codesThatKeepServerDetail = new Set(['invalid_input', 'invalid_state'])

export function errorMessage(error: unknown) {
  if (!(error instanceof ApiError)) return genericErrorMessage
  const translated = errorMessagesByCode[error.code] ?? genericErrorMessage
  const showsServerDetail = codesThatKeepServerDetail.has(error.code) && error.message !== ''
  return showsServerDetail ? `${translated} (${error.message})` : translated
}

export const roleLabels: Record<Role, string> = {
  ADMIN: 'Administrador',
  TEAM: 'Gestor de time',
  USER: 'Torcedor ou jogador',
}

export const positionLabels: Record<Position, string> = {
  GOALKEEPER: 'Goleiro',
  DEFENDER: 'Zagueiro',
  MIDFIELDER: 'Meio-campo',
  FORWARD: 'Atacante',
}

export const championshipStatusLabels: Record<ChampionshipStatus, string> = {
  OPEN: 'Inscrições abertas',
  IN_PROGRESS: 'Em andamento',
  FINISHED: 'Encerrado',
}

export const matchStatusLabels: Record<MatchStatus, string> = {
  SCHEDULED: 'Agendada',
  IN_PROGRESS: 'Ao vivo',
  FINISHED: 'Encerrada',
}

export const invitationStatusLabels: Record<InvitationStatus, string> = {
  PENDING: 'Pendente',
  ACCEPTED: 'Aceito',
  DECLINED: 'Recusado',
  CANCELED: 'Cancelado',
}

export const eventTypeLabels: Record<MatchEventType, string> = {
  GOAL: 'Gol',
  OWN_GOAL: 'Gol contra',
  SHOT: 'Chute',
  SAVE: 'Defesa',
  FOUL: 'Falta',
  YELLOW_CARD: 'Cartão amarelo',
  RED_CARD: 'Cartão vermelho',
  SUBSTITUTION: 'Substituição',
  CORNER: 'Escanteio',
  OFFSIDE: 'Impedimento',
}

export const eventTypes = Object.keys(eventTypeLabels) as MatchEventType[]
export const positions = Object.keys(positionLabels) as Position[]
