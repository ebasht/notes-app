import { useEffect, useRef, useState } from 'react'
import { createNote, deleteNote, listNotes, updateNote } from './api.js'
import { filterNotes, selectNote } from './search.js'

const blank = { id: null, title: '', body: '' }

export default function App() {
  const [notes, setNotes] = useState([])
  const [draft, setDraft] = useState(blank)
  const [saved, setSaved] = useState(blank)
  const [loading, setLoading] = useState(true)
  const [pending, setPending] = useState('')
  const [error, setError] = useState('')
  const [query, setQuery] = useState('')
  const queryRef = useRef(query)
  queryRef.current = query

  const dirty = draft.title !== saved.title || draft.body !== saved.body
  const visible = filterNotes(notes, query)

  useEffect(() => {
    let ignore = false
    listNotes()
      .then((items) => {
        if (ignore) return
        setNotes(items)
        const chosen = selectNote(items, queryRef.current)
        const next = chosen ? snapshot(chosen) : blank
        setDraft(next)
        setSaved(next)
      })
      .catch((err) => {
        if (!ignore) setError(err.message)
      })
      .finally(() => {
        if (!ignore) setLoading(false)
      })
    return () => {
      ignore = true
    }
  }, [])

  function choose(note) {
    if (!confirmDiscard()) return
    const next = snapshot(note)
    setDraft(next)
    setSaved(next)
    setError('')
  }

  function startNew() {
    if (!confirmDiscard()) return
    setDraft(blank)
    setSaved(blank)
    setError('')
  }

  function confirmDiscard() {
    if (!dirty) return true
    return window.confirm('Есть несохранённые изменения. Продолжить без сохранения?')
  }

  async function save(event) {
    event.preventDefault()
    if (!draft.title.trim()) {
      setError('Укажите заголовок')
      return
    }
    setPending('save')
    setError('')
    try {
      const payload = { title: draft.title, body: draft.body }
      const note = draft.id == null ? await createNote(payload) : await updateNote(draft.id, payload)
      setNotes((current) =>
        [note, ...current.filter((item) => item.id !== note.id)].sort(byUpdated),
      )
      const next = snapshot(note)
      setDraft(next)
      setSaved(next)
    } catch (err) {
      setError(err.message)
    } finally {
      setPending('')
    }
  }

  async function remove() {
    if (draft.id == null) return
    if (!window.confirm('Удалить заметку?')) return
    setPending('remove')
    setError('')
    try {
      await deleteNote(draft.id)
      const rest = notes.filter((item) => item.id !== draft.id)
      setNotes(rest)
      const chosen = selectNote(rest, query)
      const next = chosen ? snapshot(chosen) : blank
      setDraft(next)
      setSaved(next)
    } catch (err) {
      setError(err.message)
    } finally {
      setPending('')
    }
  }

  return (
    <main className="app">
      <header>
        <div>
          <p className="eyebrow">Личный блокнот</p>
          <h1>Заметки</h1>
          <p className="count">{countLabel(notes.length)}</p>
        </div>
        <button className="primary" type="button" onClick={startNew}>
          Новая
        </button>
      </header>

      {error ? <p className="error">{error}</p> : null}

      <section className="workspace">
        <aside className="list" aria-label="Список заметок">
          <label className="search">
            <span>Поиск</span>
            <input
              type="search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Заголовок или текст"
              autoComplete="off"
            />
          </label>
          <div className="list-scroll">
            {loading ? <p className="empty">Загрузка…</p> : null}
            {!loading && notes.length === 0 ? <p className="empty">Пока нет заметок</p> : null}
            {!loading && notes.length > 0 && visible.length === 0 ? (
              <p className="empty">Ничего не найдено</p>
            ) : null}
            {visible.map((note) => (
              <button
                key={note.id}
                type="button"
                className={note.id === draft.id ? 'note active' : 'note'}
                onClick={() => choose(note)}
              >
                <strong>{note.title}</strong>
                <span>{preview(note.body)}</span>
                <time dateTime={note.updatedAt}>{formatWhen(note.updatedAt)}</time>
              </button>
            ))}
          </div>
        </aside>

        <form className="editor" onSubmit={save}>
          <label>
            <span>Заголовок</span>
            <input
              value={draft.title}
              onChange={(event) => setDraft({ ...draft, title: event.target.value })}
              maxLength={200}
              autoComplete="off"
              placeholder="О чём заметка"
            />
          </label>
          <label className="body-field">
            <span>Текст</span>
            <textarea
              value={draft.body}
              onChange={(event) => setDraft({ ...draft, body: event.target.value })}
              placeholder="Запишите, чтобы не забыть"
            />
          </label>
          <div className="actions">
            <button className="ghost danger" type="button" onClick={remove} disabled={pending !== '' || draft.id == null}>
              {pending === 'remove' ? 'Удаление…' : 'Удалить'}
            </button>
            <div className="save-row">
              <span className="status">
                {dirty ? 'Есть несохранённые изменения' : draft.id == null ? 'Новая заметка' : 'Сохранено'}
              </span>
              <button className="primary" type="submit" disabled={pending !== '' || !dirty || !draft.title.trim()}>
                {pending === 'save' ? 'Сохранение…' : 'Сохранить'}
              </button>
            </div>
          </div>
        </form>
      </section>
    </main>
  )
}

function snapshot(note) {
  return { id: note.id, title: note.title, body: note.body }
}

function byUpdated(a, b) {
  return new Date(b.updatedAt) - new Date(a.updatedAt) || b.id - a.id
}

function preview(body) {
  const line = body.trim().split('\n').find(Boolean) || 'Пустая заметка'
  return line.length > 80 ? `${line.slice(0, 80)}…` : line
}

function formatWhen(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat('ru-RU', {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function countLabel(count) {
  const mod10 = count % 10
  const mod100 = count % 100
  if (mod10 === 1 && mod100 !== 11) return `${count} заметка`
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return `${count} заметки`
  return `${count} заметок`
}
