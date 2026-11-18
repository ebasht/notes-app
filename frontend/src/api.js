async function request(path, options) {
  let response
  try {
    response = await fetch(path, options)
  } catch {
    throw new Error('Нет связи с сервером. Запустите API на 127.0.0.1:8080.')
  }

  if (response.status === 204) {
    return null
  }

  const text = await response.text()
  let data = null
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = { error: text }
    }
  }
  if (!response.ok) {
    throw new Error(data?.error || 'Не удалось выполнить запрос')
  }
  return data
}

export function listNotes() {
  return request('/api/notes').then((data) => (Array.isArray(data) ? data : []))
}

export function createNote(note) {
  return request('/api/notes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(note),
  })
}

export function updateNote(id, note) {
  return request(`/api/notes/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(note),
  })
}

export function deleteNote(id) {
  return request(`/api/notes/${id}`, { method: 'DELETE' })
}
