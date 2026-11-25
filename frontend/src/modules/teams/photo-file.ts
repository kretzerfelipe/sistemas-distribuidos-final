const maxPhotoBytes = 2 * 1024 * 1024

export const acceptedPhotoTypes = ['image/png', 'image/jpeg', 'image/webp']

export function photoFileProblem(file: File): string | null {
  if (!acceptedPhotoTypes.includes(file.type)) return 'Use uma imagem PNG, JPEG ou WebP.'
  if (file.size > maxPhotoBytes) return 'A foto é grande demais. O limite é de 2 MB.'
  return null
}
