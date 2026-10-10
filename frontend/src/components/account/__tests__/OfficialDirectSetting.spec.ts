import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDir = dirname(fileURLToPath(import.meta.url))
const createSource = readFileSync(resolve(testDir, '../CreateAccountModal.vue'), 'utf8')
const editSource = readFileSync(resolve(testDir, '../EditAccountModal.vue'), 'utf8')

describe('official direct public account setting', () => {
  it('is opt-in on account creation', () => {
    expect(createSource).toMatch(/data-testid="public-official-direct-toggle"/)
    expect(createSource).toMatch(/const publicOfficialDirect = ref\(false\)/)
    expect(createSource).toMatch(/publicExtra\[PUBLIC_OFFICIAL_DIRECT_EXTRA_KEY\] = true/)
  })

  it('loads and persists the account-level flag while editing', () => {
    expect(editSource).toMatch(/extra\?\.\[PUBLIC_OFFICIAL_DIRECT_EXTRA_KEY\] === true/)
    expect(editSource).toMatch(/publicExtra\[PUBLIC_OFFICIAL_DIRECT_EXTRA_KEY\] = true/)
    expect(editSource).toMatch(/delete publicExtra\[PUBLIC_OFFICIAL_DIRECT_EXTRA_KEY\]/)
  })
})
