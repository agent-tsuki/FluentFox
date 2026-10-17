#!/usr/bin/env node
// Builds content/kana/*.json and content/kanji/n5.json from the FluentFoxUI
// sources, which are the current source of truth for course content.
//
//   node scripts/content/convert-frontend.mjs [path/to/fluentfoxui]
//
// The UI path defaults to ../fluentfoxui. The TypeScript sources are
// transpiled with the UI's own typescript package, so nothing is installed here.
import { createRequire } from 'node:module'
import fs from 'node:fs'
import path from 'node:path'
import vm from 'node:vm'
import { fileURLToPath } from 'node:url'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..')
const uiRoot = path.resolve(process.argv[2] ?? path.join(repoRoot, '..', 'fluentfoxui'))
const uiSrc = path.join(uiRoot, 'src')
const ts = createRequire(path.join(uiRoot, 'package.json'))('typescript')

// Kunrei / IME spellings accepted as typed answers; mirrors VARIANTS in the
// UI's speedLogic.ts, plus ぢ/づ which are typed by sound (ji/zu) or Kunrei (di/du).
const ALT_ROMAJI = {
  shi: ['si'], chi: ['ti'], tsu: ['tu'], fu: ['hu'], ji: ['zi'], wo: ['o'], n: ['nn'],
  dji: ['di', 'ji'], dzu: ['du', 'zu'],
  sha: ['sya'], shu: ['syu'], sho: ['syo'],
  cha: ['tya', 'cya'], chu: ['tyu', 'cyu'], cho: ['tyo', 'cyo'],
  jya: ['ja', 'zya'], jyu: ['ju', 'zyu'], jyo: ['jo', 'zyo'],
}

const moduleCache = new Map()

function loadTs(file) {
  if (moduleCache.has(file)) return moduleCache.get(file)
  const source = fs.readFileSync(file, 'utf8')
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  })
  const module = { exports: {} }
  moduleCache.set(file, module.exports)
  const require = (spec) => {
    if (!spec.startsWith('@/')) throw new Error(`${file}: unexpected import ${spec}`)
    return loadTs(path.join(uiSrc, spec.slice(2) + '.ts'))
  }
  vm.runInNewContext(outputText, { module, exports: module.exports, require })
  moduleCache.set(file, module.exports)
  return module.exports
}

function writeJson(rel, data) {
  const file = path.join(repoRoot, 'content', rel)
  fs.mkdirSync(path.dirname(file), { recursive: true })
  fs.writeFileSync(file, JSON.stringify(data, null, 2) + '\n')
  console.log(`wrote ${path.relative(repoRoot, file)}`)
}

function strokeCount(strokes, character) {
  let total = 0
  for (const ch of character) {
    if (!strokes[ch]) return undefined
    total += strokes[ch].strokes.length
  }
  return total
}

// Pairs are stored one way per row, so each unordered pair becomes two rows.
function addLookAlike(map, from, to, note) {
  if (!from || !to || from === to) return
  const list = (map[from] ??= [])
  const existing = list.find((l) => l.slug === to)
  if (existing) {
    if (!existing.note && note) existing.note = note
    return
  }
  list.push(note ? { slug: to, note } : { slug: to })
}

function buildKana(script, { sections, units, groups, sets, charMap }) {
  const chars = new Map()
  for (const section of sections) for (const c of section.chars) if (!c.isEmpty) chars.set(c.id, c)
  const strokes = JSON.parse(fs.readFileSync(path.join(uiSrc, 'assets', 'kana-strokes', `${script}.json`), 'utf8'))
  const slugByChar = new Map([...chars.values()].map((c) => [c.character, c.id]))
  const groupOfUnit = new Map(groups.flatMap((g) => g.unitIds.map((u) => [u, g.id])))

  const lookAlikes = {}
  for (const set of sets) for (const a of set.ids) for (const b of set.ids) addLookAlike(lookAlikes, a, b, set.tip)
  for (const [fromChar, toChars] of Object.entries(charMap)) {
    for (const toChar of toChars) {
      const from = slugByChar.get(fromChar)
      const to = slugByChar.get(toChar)
      if (!from || !to) console.warn(`${script}: skipped look-alike ${fromChar} → ${toChar}`)
      addLookAlike(lookAlikes, from, to)
    }
  }

  const kana = []
  units.forEach((unit) => {
    for (const id of unit.kanaIds) {
      const c = chars.get(id)
      if (!c) throw new Error(`${script}: unit ${unit.id} lists unknown kana ${id}`)
      kana.push({
        slug: c.id,
        character: c.character,
        romaji: c.romaji,
        alt_romaji: ALT_ROMAJI[c.id] ?? [],
        group: groupOfUnit.get(unit.id),
        unit: unit.id,
        position: kana.length + 1,
        stroke_count: strokeCount(strokes, c.character),
        look_alikes: lookAlikes[c.id] ?? [],
      })
    }
  })
  const taught = new Set(kana.map((k) => k.slug))
  for (const id of chars.keys()) if (!taught.has(id)) console.warn(`${script}: chart kana ${id} is in no unit, skipped`)

  return {
    script,
    groups: groups.map((g, i) => ({ slug: g.id, title: g.title, summary: g.summary, position: i + 1 })),
    units: units.map((u, i) => ({
      slug: u.id,
      group: groupOfUnit.get(u.id),
      title: u.title,
      summary: u.summary,
      position: i + 1,
      youtube_id: u.youtubeId ?? null,
      video_credit: u.videoCredit ?? null,
    })),
    kana,
  }
}

function lookAlikeMapFromSets(sets) {
  const out = {}
  for (const { chars } of sets) for (const a of chars) for (const b of chars) if (a !== b) (out[a] ??= []).push(b)
  return out
}

function convertKana() {
  const kanaDir = path.join(uiSrc, 'lib', 'content', 'kana')
  const hiraganaUnits = loadTs(path.join(kanaDir, 'hiraganaUnits.ts'))
  const katakanaUnits = loadTs(path.join(kanaDir, 'katakanaUnits.ts'))
  const { HIRAGANA_LOOK_ALIKE_SETS } = loadTs(path.join(kanaDir, 'lookAlikeSets.ts'))
  const { HIRAGANA_LOOK_ALIKES } = loadTs(path.join(kanaDir, 'hiraganaLookAlikes.ts'))
  const { KATAKANA_LOOK_ALIKES } = loadTs(path.join(kanaDir, 'katakanaLookAlikes.ts'))
  const { mockHiraganaSections } = loadTs(path.join(uiSrc, 'api', 'mock', 'hiragana.ts'))
  const { mockKatakanaSections } = loadTs(path.join(uiSrc, 'api', 'mock', 'katakana.ts'))

  writeJson('kana/hiragana.json', buildKana('hiragana', {
    sections: mockHiraganaSections,
    units: hiraganaUnits.HIRAGANA_UNITS,
    groups: hiraganaUnits.HIRAGANA_GROUPS,
    sets: HIRAGANA_LOOK_ALIKE_SETS,
    charMap: HIRAGANA_LOOK_ALIKES,
  }))
  // Same extras as the UI's katakana lookAlikeMap: small ッ vs big ツ.
  const katakanaCharMap = { ...lookAlikeMapFromSets(KATAKANA_LOOK_ALIKES) }
  katakanaCharMap['ッ'] = [...(katakanaCharMap['ッ'] ?? []), 'ツ', 'シ']
  katakanaCharMap['ツ'] = [...(katakanaCharMap['ツ'] ?? []), 'ッ']
  writeJson('kana/katakana.json', buildKana('katakana', {
    sections: mockKatakanaSections,
    units: katakanaUnits.KATAKANA_UNITS,
    groups: katakanaUnits.KATAKANA_GROUPS,
    sets: KATAKANA_LOOK_ALIKES,
    charMap: katakanaCharMap,
  }))
}

function convertKanji(levelId) {
  const dir = path.join(uiSrc, 'lib', 'content', 'kanji', 'generated')
  const index = JSON.parse(fs.readFileSync(path.join(dir, `${levelId}.index.json`), 'utf8'))
  const detail = JSON.parse(fs.readFileSync(path.join(dir, `${levelId}.detail.json`), 'utf8'))
  const groupOfUnit = new Map(index.groups.flatMap((g) => g.unitIds.map((u) => [u, g.id])))

  const parts = new Map()
  for (const e of detail.entries) for (const p of e.parts) if (!parts.has(p.char)) parts.set(p.char, p.name)

  const entries = detail.entries.map((e, i) => {
    return {
      char: e.char,
      unit: e.unitId,
      position: i + 1,
      meanings: e.meanings,
      synonyms: e.synonyms ?? [],
      onyomi: e.on,
      kun: e.kun.map((k) => ({ kana: k.kana, okurigana: k.okurigana ?? null, affix: k.affix ?? null })),
      key_reading: { type: e.keyReading.type, kana: e.keyReading.kana },
      reading_tip: e.readingTip ?? null,
      mnemonic: e.mnemonic ? { text: e.mnemonic.text, draft: e.mnemonic.draft } : null,
      strokes: e.strokes,
      grade: e.grade ?? null,
      freq: e.freq ?? null,
      parts: e.parts.map((p) => p.char),
      vocab: e.vocab.map((v) => ({ word: v.word, reading: v.reading, meaning: v.meaning, is_key: v.isKey === true })),
      look_alikes: (e.lookAlikes ?? []).map((l) => ({ char: l.char, note: l.note ?? null })),
    }
  })

  writeJson(`kanji/${levelId}.json`, {
    level: levelId.toUpperCase(),
    groups: index.groups.map((g, i) => ({ slug: g.id, title: g.title, summary: g.summary, position: i + 1 })),
    units: index.units.map((u, i) => ({
      slug: u.id,
      group: groupOfUnit.get(u.id),
      title: u.title,
      summary: u.summary,
      position: i + 1,
    })),
    parts: [...parts].map(([char, name]) => ({ char, name })),
    entries,
  })
}

convertKana()
convertKanji('n5')
