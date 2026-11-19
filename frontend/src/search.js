export function filterNotes(notes, query) {
  const needle = query.trim().toLowerCase()
  if (needle === '') return notes
  return notes.filter((note) => {
    const title = note.title.toLowerCase()
    const body = note.body.toLowerCase()
    return title.includes(needle) || body.includes(needle)
  })
}

export function selectNote(notes, query) {
  return filterNotes(notes, query)[0] ?? null
}
