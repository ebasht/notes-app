import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { filterNotes } from './search.js'

const notes = [
  { id: 1, title: 'Масло', body: '5w30' },
  { id: 2, title: 'Фильтр', body: 'замена масла' },
  { id: 3, title: 'a.*b', body: '5w30 (OEM)' },
]

describe('filterNotes', () => {
  it('returns the same list and order for an empty or blank query', () => {
    assert.equal(filterNotes(notes, ''), notes)
    assert.equal(filterNotes(notes, '   \n\t'), notes)
    assert.deepEqual(
      filterNotes(notes, '').map((note) => note.id),
      [1, 2, 3],
    )
  })

  it('matches a title case-insensitively', () => {
    const found = filterNotes(notes, '  масло  ')
    assert.deepEqual(found.map((note) => note.id), [1])
    assert.equal(found[0].title, 'Масло')
  })

  it('matches the body when the title does not', () => {
    const found = filterNotes(notes, 'замена')
    assert.deepEqual(found.map((note) => note.id), [2])
    assert.equal(found[0].title, 'Фильтр')
  })

  it('matches a substring rather than a whole word', () => {
    const found = filterNotes(notes, 'мас')
    assert.deepEqual(found.map((note) => note.id), [1, 2])
  })

  it('treats regular-expression characters as plain text', () => {
    assert.deepEqual(filterNotes(notes, '.*').map((note) => note.id), [3])
    assert.deepEqual(filterNotes(notes, '(OEM)').map((note) => note.id), [3])
    assert.deepEqual(filterNotes(notes, '5w30.'), [])
  })

  it('returns an empty array when nothing matches', () => {
    assert.deepEqual(filterNotes(notes, 'антифриз'), [])
  })

  it('does not mutate the original array', () => {
    const snapshot = notes.map((note) => ({ ...note }))
    filterNotes(notes, 'масло')
    filterNotes(notes, '')
    filterNotes(notes, '.*')
    assert.deepEqual(notes, snapshot)
    assert.equal(notes.length, 3)
  })
})
