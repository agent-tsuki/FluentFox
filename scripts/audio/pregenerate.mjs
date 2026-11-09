#!/usr/bin/env node
/**
 * Generates the spoken clips for the course's chapter content ahead of time,
 * so learners hear the recorded voice at once instead of the browser's voice
 * (guests can only play clips that already exist).
 *
 * It asks the API (POST /audio/speech) exactly as the app does, with the same
 * texts: listening lines as written (readings and spaces removed), each in its
 * character's voice (the passage's speakers name a cast key), and chapter
 * words as kana in the main voice. The
 * API makes a few new clips per request, so batches are repeated until every
 * clip exists. Use an admin account: admins have no hourly limit.
 *
 *   FOXSENSEI_TOKEN=<admin access token> node scripts/audio/pregenerate.mjs [--api URL] [--only listening|words] [--dry-run]
 *
 * The token is read from the environment for this one command; it is never
 * printed or saved. Re-running is safe: existing clips are only looked up.
 */
import { readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'

const args = process.argv.slice(2)
const flag = (name, fallback) => {
  const i = args.indexOf(`--${name}`)
  return i === -1 ? fallback : args[i + 1]
}
const API = flag('api', 'https://api.foxsensei.com').replace(/\/+$/, '')
const ONLY = flag('only', 'all')
const DRY_RUN = args.includes('--dry-run')
const TOKEN = process.env.FOXSENSEI_TOKEN
if (!TOKEN && !DRY_RUN) {
  console.error('Set FOXSENSEI_TOKEN to an admin access token for this command.')
  process.exit(1)
}

/** Must match the API's per-request limit (common.AUDIO_MAX_TEXTS). */
const BATCH = 20
/** Rounds per batch before giving up on it (each round makes a few clips). */
const MAX_ROUNDS = 40
const WAIT_MS = 1500

const contentDir = join(import.meta.dirname, '../../content')
const readAll = (dir) => readdirSync(join(contentDir, dir)).filter((f) => f.endsWith('.json')).map((f) => JSON.parse(readFileSync(join(contentDir, dir, f), 'utf8')))

/** "私[わたし] は 学生[がくせい] です。" -> "私は学生です。" (what the Listening tab speaks). */
const spoken = (markup) => markup.replace(/\[[^\]]*\]/g, '').replace(/\s+/g, '')

/** Texts by voice ("main", or a cast key such as f1). */
const byVoice = new Map()
const add = (voice, text) => {
  if (!byVoice.has(voice)) byVoice.set(voice, new Set())
  byVoice.get(voice).add(text)
}
if (ONLY === 'all' || ONLY === 'listening') {
  for (const file of readAll('listening')) {
    for (const chapter of file.chapters) {
      for (const passage of chapter.passages) {
        for (const line of passage.lines) {
          // A dialogue line is said by its speaker; a monologue by its reader.
          const speaker = passage.speakers[line.speaker ?? 0]
          add(speaker?.voice ?? 'main', spoken(line.text))
        }
      }
    }
  }
}
if (ONLY === 'all' || ONLY === 'words') {
  for (const file of readAll('chapters')) {
    for (const chapter of file.chapters) for (const word of chapter.words) add('main', word.kana)
  }
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

async function speak(texts, voice) {
  const res = await fetch(`${API}/audio/speech`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${TOKEN}` },
    body: JSON.stringify(voice === 'main' ? { texts } : { texts, voice }),
  })
  if (res.status === 401 || res.status === 403) throw new Error(`the API refused the token (${res.status}); sign in again and use a fresh admin access token`)
  if (!res.ok) throw new Error(`POST /audio/speech answered ${res.status}`)
  return (await res.json()).data
}

async function generate(texts, voice) {
  const list = [...texts]
  let done = 0
  let failed = 0
  for (let i = 0; i < list.length; i += BATCH) {
    const batch = list.slice(i, i + BATCH)
    let pending = true
    for (let round = 0; round < MAX_ROUNDS && pending; round++) {
      const result = await speak(batch, voice)
      pending = result.pending
      if (pending) await sleep(WAIT_MS)
    }
    if (pending) failed += batch.length
    else done += batch.length
    process.stdout.write(`\r${voice} voice: ${done + failed}/${list.length} texts${failed ? ` (${failed} still pending)` : ''}   `)
  }
  process.stdout.write('\n')
  return failed
}

const voices = [...byVoice.keys()].sort()
console.log(`${API}: ${voices.map((v) => `${v} ${byVoice.get(v).size}`).join(', ')} texts`)
if (DRY_RUN) process.exit(0)
let failed = 0
for (const voice of voices) failed += await generate(byVoice.get(voice), voice)
if (failed) {
  console.log(`${failed} texts still have no clip (the voice engine may be busy or down). Run the script again later.`)
  process.exit(2)
}
console.log('Every clip exists.')
